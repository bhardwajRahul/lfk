package k8s

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildHelmReleaseSecretData synthesizes a helm release secret payload in the
// exact format helm writes it: base64(gzip(json(blob))).
func buildHelmReleaseSecretData(t *testing.T, blob helmReleaseBlob) []byte {
	t.Helper()
	raw, err := json.Marshal(blob)
	require.NoError(t, err)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err = gw.Write(raw)
	require.NoError(t, err)
	require.NoError(t, gw.Close())

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return []byte(encoded)
}

func TestDecodeHelmReleaseSecret_HappyPath(t *testing.T) {
	blob := helmReleaseBlob{
		Name:      "my-release",
		Version:   7,
		Namespace: "default",
	}
	blob.Chart.Metadata.Name = "nginx"
	blob.Chart.Metadata.Version = "15.4.2"
	blob.Chart.Metadata.AppVersion = "1.25.3"
	blob.Info.Status = "deployed"
	blob.Info.Description = "Upgrade complete"
	blob.Info.LastDeployed = "2024-05-01T10:11:12Z"

	data := buildHelmReleaseSecretData(t, blob)

	info, err := decodeHelmReleaseSecret(data)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 7, info.Revision)
	assert.Equal(t, "deployed", info.Status)
	assert.Equal(t, "nginx", info.ChartName)
	assert.Equal(t, "15.4.2", info.ChartVersion)
	assert.Equal(t, "1.25.3", info.AppVersion)
	assert.Equal(t, "Upgrade complete", info.Description)
	assert.False(t, info.LastDeployed.IsZero())
}

func TestDecodeHelmReleaseSecret_InvalidBase64(t *testing.T) {
	info, err := decodeHelmReleaseSecret([]byte("!!!not-base64!!!"))
	require.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "base64 decode")
}

func TestDecodeHelmReleaseSecret_InvalidGzip(t *testing.T) {
	// Valid base64 but not a gzip stream.
	payload := base64.StdEncoding.EncodeToString([]byte("not a gzip payload"))
	info, err := decodeHelmReleaseSecret([]byte(payload))
	require.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "gzip reader")
}

func TestDecodeHelmReleaseSecret_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write([]byte("this is not json"))
	require.NoError(t, err)
	require.NoError(t, gw.Close())

	payload := base64.StdEncoding.EncodeToString(buf.Bytes())
	info, err := decodeHelmReleaseSecret([]byte(payload))
	require.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "unmarshal release")
}

func TestDecodeHelmReleaseSecret_Empty(t *testing.T) {
	info, err := decodeHelmReleaseSecret(nil)
	require.Error(t, err)
	assert.Nil(t, info)
}

func TestDecodeHelmReleaseSecret_SizeLimit(t *testing.T) {
	// Build a payload that decompresses into more than the configured max.
	// A 51 MiB payload of repeating bytes compresses to a few KiB but expands
	// back past the 50 MiB limit. Using 51 MiB (not 60) avoids needlessly
	// stressing memory-constrained CI runners.
	huge := strings.Repeat("A", 51*1024*1024)
	raw, err := json.Marshal(map[string]string{"payload": huge})
	require.NoError(t, err)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err = gw.Write(raw)
	require.NoError(t, err)
	require.NoError(t, gw.Close())

	payload := base64.StdEncoding.EncodeToString(buf.Bytes())
	info, err := decodeHelmReleaseSecret([]byte(payload))
	require.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "limit exceeded")
}

func TestDecodeHelmReleaseSecret_UnparseableTimestamp(t *testing.T) {
	// LastDeployed in an unexpected format should not fail the decode; the
	// resulting info is returned with a zero LastDeployed so the caller can
	// render the rest of the release fields.
	blob := helmReleaseBlob{Name: "weird", Version: 2}
	blob.Info.LastDeployed = "2024-01-15 10:00:00.000" // not RFC3339
	data := buildHelmReleaseSecretData(t, blob)

	info, err := decodeHelmReleaseSecret(data)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 2, info.Revision)
	assert.True(t, info.LastDeployed.IsZero(), "unparseable timestamp must leave LastDeployed zero")
}

func TestDecodeHelmReleaseSecret_MissingFieldsGraceful(t *testing.T) {
	// Only name + version set; chart/info omitted entirely.
	blob := helmReleaseBlob{Name: "bare", Version: 1}
	data := buildHelmReleaseSecretData(t, blob)

	info, err := decodeHelmReleaseSecret(data)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 1, info.Revision)
	assert.Empty(t, info.ChartName)
	assert.Empty(t, info.ChartVersion)
	assert.Empty(t, info.AppVersion)
	assert.True(t, info.LastDeployed.IsZero())
}

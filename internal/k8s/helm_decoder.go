package k8s

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// helmReleaseMaxBytes is the maximum decompressed size accepted from a helm
// release secret blob. This is a safeguard against decompression bombs.
const helmReleaseMaxBytes int64 = 50 * 1024 * 1024 // 50 MiB

// helmReleaseBlob mirrors the subset of fields of a Helm v3 release that we
// need to render in the list and details views. The helm release struct has
// many more fields, but defining only what we use keeps the decoder tiny and
// avoids pulling in the helm SDK as a dependency.
type helmReleaseBlob struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
	Chart   struct {
		Metadata struct {
			Name       string `json:"name"`
			Version    string `json:"version"`
			AppVersion string `json:"appVersion"`
		} `json:"metadata"`
	} `json:"chart"`
	Info struct {
		Status       string `json:"status"`
		Description  string `json:"description"`
		LastDeployed string `json:"last_deployed"`
	} `json:"info"`
	Namespace string `json:"namespace"`
	Manifest  string `json:"manifest"`
}

// HelmReleaseInfo is the decoded, user-friendly view of a helm release blob.
type HelmReleaseInfo struct {
	Revision     int
	Status       string
	ChartName    string
	ChartVersion string
	AppVersion   string
	Description  string
	LastDeployed time.Time
	Manifest     string
}

// decodeHelmReleaseSecret decodes the Data["release"] value of a helm secret
// and returns a HelmReleaseInfo. Helm stores releases as base64(gzip(json(blob))),
// so the decode pipeline reverses those three steps with a hard size cap on
// the gzip reader to guard against decompression bombs.
func decodeHelmReleaseSecret(data []byte) (*HelmReleaseInfo, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("decoding helm release: empty input")
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("decoding helm release: base64 decode: %w", err)
	}

	gr, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return nil, fmt.Errorf("decoding helm release: gzip reader: %w", err)
	}

	// Read at most helmReleaseMaxBytes+1 to detect overflow without loading more.
	limited := io.LimitReader(gr, helmReleaseMaxBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		_ = gr.Close()
		return nil, fmt.Errorf("decoding helm release: gzip read: %w", err)
	}
	if int64(len(raw)) > helmReleaseMaxBytes {
		_ = gr.Close()
		return nil, fmt.Errorf("decoding helm release: limit exceeded (>%d bytes)", helmReleaseMaxBytes)
	}
	// Close() validates the gzip CRC32 checksum; a truncated or corrupt blob
	// would otherwise decode silently into a structurally valid but wrong info.
	if err := gr.Close(); err != nil {
		return nil, fmt.Errorf("decoding helm release: gzip checksum: %w", err)
	}

	var blob helmReleaseBlob
	if err := json.Unmarshal(raw, &blob); err != nil {
		return nil, fmt.Errorf("decoding helm release: unmarshal release: %w", err)
	}

	info := &HelmReleaseInfo{
		Revision:     blob.Version,
		Status:       blob.Info.Status,
		ChartName:    blob.Chart.Metadata.Name,
		ChartVersion: blob.Chart.Metadata.Version,
		AppVersion:   blob.Chart.Metadata.AppVersion,
		Description:  blob.Info.Description,
		Manifest:     blob.Manifest,
	}
	if blob.Info.LastDeployed != "" {
		if t, perr := time.Parse(time.RFC3339Nano, blob.Info.LastDeployed); perr == nil {
			info.LastDeployed = t
		} else if t, perr := time.Parse(time.RFC3339, blob.Info.LastDeployed); perr == nil {
			info.LastDeployed = t
		}
	}
	return info, nil
}

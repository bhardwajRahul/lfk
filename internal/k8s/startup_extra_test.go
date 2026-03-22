package k8s

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

// --- computeImagePullTime: missing branch (negative duration ignored) ---

func TestComputeImagePullTime_NegativeDuration(t *testing.T) {
	// When the "Pulled" event has an earlier timestamp than "Pulling",
	// the negative duration should be ignored (line 283: d > 0 check).
	now := time.Now()
	events := []corev1.Event{
		{
			Reason:        "Pulling",
			Message:       `Pulling image "nginx:latest"`,
			LastTimestamp: metav1.Time{Time: now.Add(5 * time.Second)},
		},
		{
			Reason:        "Pulled",
			Message:       `Successfully pulled image "nginx:latest"`,
			LastTimestamp: metav1.Time{Time: now},
		},
	}
	result := computeImagePullTime(events)
	assert.Equal(t, time.Duration(0), result,
		"negative pull duration should be ignored")
}

func TestComputeImagePullTime_PulledWithoutPulling(t *testing.T) {
	// A "Pulled" event without a matching "Pulling" event should still
	// create a pullPair entry but result in zero duration since pulling
	// time is zero (line 281: !pair.pulling.IsZero() fails).
	now := time.Now()
	events := []corev1.Event{
		{
			Reason:        "Pulled",
			Message:       `Successfully pulled image "redis:7"`,
			LastTimestamp: metav1.Time{Time: now},
		},
	}
	result := computeImagePullTime(events)
	assert.Equal(t, time.Duration(0), result)
}

func TestComputeImagePullTime_EmptyImageInMessage(t *testing.T) {
	// Events with messages that don't contain quoted image names
	// should result in empty image string and be ignored.
	now := time.Now()
	events := []corev1.Event{
		{
			Reason:        "Pulling",
			Message:       "Pulling image without quotes",
			LastTimestamp: metav1.Time{Time: now},
		},
		{
			Reason:        "Pulled",
			Message:       "Successfully pulled image without quotes",
			LastTimestamp: metav1.Time{Time: now.Add(1 * time.Second)},
		},
	}
	result := computeImagePullTime(events)
	assert.Equal(t, time.Duration(0), result)
}

func TestComputeImagePullTime_MixedPulledAndUnpulled(t *testing.T) {
	// One image has both Pulling and Pulled, another only has Pulling.
	now := time.Now()
	events := []corev1.Event{
		{
			Reason:        "Pulling",
			Message:       `Pulling image "nginx:latest"`,
			LastTimestamp: metav1.Time{Time: now},
		},
		{
			Reason:        "Pulled",
			Message:       `Successfully pulled image "nginx:latest"`,
			LastTimestamp: metav1.Time{Time: now.Add(2 * time.Second)},
		},
		{
			Reason:        "Pulling",
			Message:       `Pulling image "redis:7"`,
			LastTimestamp: metav1.Time{Time: now.Add(3 * time.Second)},
		},
	}
	result := computeImagePullTime(events)
	assert.Equal(t, 2*time.Second, result,
		"only nginx pull (2s) should be counted; redis has no Pulled event")
}

package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

// --- containerStatusFromPod ---

func TestContainerStatusFromPod(t *testing.T) {
	tests := []struct {
		name     string
		cName    string
		statuses []corev1.ContainerStatus
		want     string
	}{
		{
			name:  "found running and ready",
			cName: "app",
			statuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: true,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
			want: "Running",
		},
		{
			name:  "found running but not ready",
			cName: "app",
			statuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: false,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
			want: "NotReady",
		},
		{
			name:  "found waiting",
			cName: "app",
			statuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: false,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
					},
				},
			},
			want: "Waiting",
		},
		{
			name:  "found terminated completed",
			cName: "app",
			statuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: false,
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{Reason: "Completed"},
					},
				},
			},
			want: "Completed",
		},
		{
			name:  "found terminated other reason",
			cName: "app",
			statuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: false,
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 1},
					},
				},
			},
			want: "Terminated",
		},
		{
			name:  "found unknown state",
			cName: "app",
			statuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: false,
					State: corev1.ContainerState{},
				},
			},
			want: "Unknown",
		},
		{
			name:  "not found in statuses",
			cName: "missing",
			statuses: []corev1.ContainerStatus{
				{
					Name:  "other",
					Ready: true,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
			want: "Waiting",
		},
		{
			name:     "nil statuses",
			cName:    "app",
			statuses: nil,
			want:     "Waiting",
		},
		{
			name:     "empty statuses",
			cName:    "app",
			statuses: []corev1.ContainerStatus{},
			want:     "Waiting",
		},
		{
			name:  "picks correct container among multiple",
			cName: "sidecar",
			statuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: true,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
				{
					Name:  "sidecar",
					Ready: false,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "ContainerCreating"},
					},
				},
			},
			want: "Waiting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containerStatusFromPod(tt.cName, tt.statuses)
			assert.Equal(t, tt.want, got)
		})
	}
}

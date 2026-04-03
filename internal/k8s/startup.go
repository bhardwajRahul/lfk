package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodStartupInfo holds the timing breakdown of a pod's startup sequence.
type PodStartupInfo struct {
	PodName   string
	Namespace string
	TotalTime time.Duration
	Phases    []StartupPhase
}

// StartupPhase represents a single phase in the pod startup sequence.
type StartupPhase struct {
	Name     string
	Duration time.Duration
	Status   string // "completed", "in-progress", "unknown"
}

// podConditionTimes holds parsed condition timestamps from a pod's status.
type podConditionTimes struct {
	scheduled       time.Time
	initialized     time.Time
	containersReady time.Time
	ready           time.Time
}

// GetPodStartupAnalysis fetches a pod and its events to compute a startup timing breakdown.
func (c *Client) GetPodStartupAnalysis(ctx context.Context, contextName, namespace, podName string) (*PodStartupInfo, error) {
	clientset, err := c.clientsetForContext(contextName)
	if err != nil {
		return nil, err
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting pod: %w", err)
	}

	info := &PodStartupInfo{
		PodName:   podName,
		Namespace: namespace,
	}

	creationTime := pod.CreationTimestamp.Time
	now := time.Now()

	ct := extractConditionTimes(pod.Status.Conditions)

	// Phase 1: Scheduling.
	appendSchedulingPhase(info, ct.scheduled, creationTime, now)

	// Phase 2: Image Pull (from events).
	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", podName),
	})
	if err == nil && events != nil {
		appendImagePullPhase(info, events.Items, now)
	}

	// Phase 3: Init Containers.
	if len(pod.Spec.InitContainers) > 0 {
		appendInitContainerPhases(info, ct, pod.Status.InitContainerStatuses, now)
	}

	// Phase 4: Container Startup.
	appendContainerStartupPhases(info, ct, pod.Status.ContainerStatuses, now)

	// Phase 5: Readiness.
	appendReadinessPhase(info, ct.containersReady, ct.ready, now)

	// Compute total time.
	info.TotalTime = computeTotalStartupTime(creationTime, ct, now)

	return info, nil
}

// extractConditionTimes parses condition timestamps from a pod's status conditions.
func extractConditionTimes(conditions []corev1.PodCondition) podConditionTimes {
	var ct podConditionTimes
	for _, cond := range conditions {
		if cond.LastTransitionTime.IsZero() {
			continue
		}
		switch cond.Type {
		case "PodScheduled":
			ct.scheduled = cond.LastTransitionTime.Time
		case "Initialized":
			ct.initialized = cond.LastTransitionTime.Time
		case "ContainersReady":
			ct.containersReady = cond.LastTransitionTime.Time
		case "Ready":
			ct.ready = cond.LastTransitionTime.Time
		}
	}
	return ct
}

// appendSchedulingPhase adds the scheduling phase (Created -> PodScheduled).
func appendSchedulingPhase(info *PodStartupInfo, scheduledTime, creationTime, now time.Time) {
	if !scheduledTime.IsZero() {
		info.Phases = append(info.Phases, StartupPhase{
			Name:     "Scheduling",
			Duration: scheduledTime.Sub(creationTime),
			Status:   "completed",
		})
	} else {
		info.Phases = append(info.Phases, StartupPhase{
			Name:     "Scheduling",
			Duration: now.Sub(creationTime),
			Status:   "in-progress",
		})
	}
}

// appendImagePullPhase adds the image pull phase from events.
func appendImagePullPhase(info *PodStartupInfo, events []corev1.Event, now time.Time) {
	pullDuration := computeImagePullTime(events)
	if pullDuration > 0 {
		info.Phases = append(info.Phases, StartupPhase{
			Name:     "Image Pull",
			Duration: pullDuration,
			Status:   "completed",
		})
		return
	}
	// Check if pulling is in progress.
	for _, ev := range events {
		if ev.Reason == "Pulling" {
			info.Phases = append(info.Phases, StartupPhase{
				Name:     "Image Pull",
				Duration: now.Sub(ev.LastTimestamp.Time),
				Status:   "in-progress",
			})
			break
		}
	}
}

// appendInitContainerPhases adds the overall init container phase and per-container timing.
func appendInitContainerPhases(info *PodStartupInfo, ct podConditionTimes, statuses []corev1.ContainerStatus, now time.Time) {
	if !ct.initialized.IsZero() && !ct.scheduled.IsZero() {
		info.Phases = append(info.Phases, StartupPhase{
			Name:     "Init Containers",
			Duration: ct.initialized.Sub(ct.scheduled),
			Status:   "completed",
		})
	} else if !ct.scheduled.IsZero() {
		info.Phases = append(info.Phases, StartupPhase{
			Name:     "Init Containers",
			Duration: now.Sub(ct.scheduled),
			Status:   "in-progress",
		})
	}

	// Add per-init-container timing if available.
	for _, cs := range statuses {
		info.Phases = append(info.Phases, containerStatusToPhase(cs, "init", now, time.Time{}))
	}
}

// appendContainerStartupPhases adds the overall container startup phase and per-container timing.
func appendContainerStartupPhases(info *PodStartupInfo, ct podConditionTimes, statuses []corev1.ContainerStatus, now time.Time) {
	baseTime := ct.initialized
	if baseTime.IsZero() {
		baseTime = ct.scheduled
	}
	if !ct.containersReady.IsZero() && !baseTime.IsZero() {
		info.Phases = append(info.Phases, StartupPhase{
			Name:     "Container Startup",
			Duration: ct.containersReady.Sub(baseTime),
			Status:   "completed",
		})
	} else if !baseTime.IsZero() {
		info.Phases = append(info.Phases, StartupPhase{
			Name:     "Container Startup",
			Duration: now.Sub(baseTime),
			Status:   "in-progress",
		})
	}

	// Add per-container timing.
	for _, cs := range statuses {
		info.Phases = append(info.Phases, containerStatusToPhase(cs, "container", now, ct.containersReady))
	}
}

// containerStatusToPhase converts a container status into a startup phase entry.
// prefix is "init" or "container". endTime is used as the end of a running container's
// phase (containersReadyTime for regular containers, zero for init containers).
func containerStatusToPhase(cs corev1.ContainerStatus, prefix string, now, endTime time.Time) StartupPhase {
	phaseName := fmt.Sprintf("  %s: %s", prefix, cs.Name)

	switch {
	case cs.State.Terminated != nil:
		start := cs.State.Terminated.StartedAt.Time
		finish := cs.State.Terminated.FinishedAt.Time
		if !start.IsZero() && !finish.IsZero() {
			return StartupPhase{Name: phaseName, Duration: finish.Sub(start), Status: "completed"}
		}
		return StartupPhase{Name: phaseName, Duration: 0, Status: "unknown"}
	case cs.State.Running != nil:
		startedAt := cs.State.Running.StartedAt.Time
		if !startedAt.IsZero() && !endTime.IsZero() {
			return StartupPhase{Name: phaseName, Duration: endTime.Sub(startedAt), Status: "completed"}
		}
		if !startedAt.IsZero() {
			return StartupPhase{Name: phaseName, Duration: now.Sub(startedAt), Status: "in-progress"}
		}
		return StartupPhase{Name: phaseName, Duration: 0, Status: "unknown"}
	default:
		return StartupPhase{Name: phaseName, Duration: 0, Status: "unknown"}
	}
}

// appendReadinessPhase adds the readiness probe phase (ContainersReady -> Ready).
func appendReadinessPhase(info *PodStartupInfo, containersReadyTime, readyTime, now time.Time) {
	if !readyTime.IsZero() && !containersReadyTime.IsZero() {
		readinessDur := readyTime.Sub(containersReadyTime)
		if readinessDur > 0 {
			info.Phases = append(info.Phases, StartupPhase{
				Name:     "Readiness Probes",
				Duration: readinessDur,
				Status:   "completed",
			})
		}
	} else if !containersReadyTime.IsZero() && readyTime.IsZero() {
		info.Phases = append(info.Phases, StartupPhase{
			Name:     "Readiness Probes",
			Duration: now.Sub(containersReadyTime),
			Status:   "in-progress",
		})
	}
}

// computeTotalStartupTime computes the total startup time from creation to ready.
func computeTotalStartupTime(creationTime time.Time, ct podConditionTimes, now time.Time) time.Duration {
	switch {
	case !ct.ready.IsZero():
		return ct.ready.Sub(creationTime)
	case !ct.containersReady.IsZero():
		return ct.containersReady.Sub(creationTime)
	default:
		return now.Sub(creationTime)
	}
}

// computeImagePullTime calculates total image pull duration from events.
// It pairs "Pulling" and "Pulled" events for each image and sums up the durations.
func computeImagePullTime(events []corev1.Event) time.Duration {
	type pullPair struct {
		pulling time.Time
		pulled  time.Time
	}

	// Group by image name (extracted from the event message).
	pulls := make(map[string]*pullPair)

	// Sort events by timestamp to process them in order.
	sort.Slice(events, func(i, j int) bool {
		return events[i].LastTimestamp.Time.Before(events[j].LastTimestamp.Time)
	})

	for _, ev := range events {
		switch ev.Reason {
		case "Pulling":
			image := extractImageFromMessage(ev.Message)
			if image != "" {
				if _, ok := pulls[image]; !ok {
					pulls[image] = &pullPair{}
				}
				pulls[image].pulling = ev.LastTimestamp.Time
			}
		case "Pulled":
			image := extractImageFromMessage(ev.Message)
			if image != "" {
				if _, ok := pulls[image]; !ok {
					pulls[image] = &pullPair{}
				}
				pulls[image].pulled = ev.LastTimestamp.Time
			}
		}
	}

	var total time.Duration
	for _, pair := range pulls {
		if !pair.pulling.IsZero() && !pair.pulled.IsZero() {
			d := pair.pulled.Sub(pair.pulling)
			if d > 0 {
				total += d
			}
		}
	}
	return total
}

// extractImageFromMessage extracts an image name from a Pulling/Pulled event message.
// Typical formats: "Pulling image \"nginx:latest\"" or "Successfully pulled image \"nginx:latest\""
func extractImageFromMessage(message string) string {
	// Look for content between quotes.
	start := strings.Index(message, "\"")
	if start < 0 {
		return ""
	}
	end := strings.Index(message[start+1:], "\"")
	if end < 0 {
		return ""
	}
	return message[start+1 : start+1+end]
}

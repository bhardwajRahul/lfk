package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/janosmiko/lfk/internal/model"
)

// DiscoverCRDs lists all installed CRDs on the given cluster context and returns
// them as ResourceTypeEntry values suitable for navigation. Each CRD is mapped
// to a resource type using its spec metadata (group, versions, names, scope).
func (c *Client) DiscoverCRDs(ctx context.Context, contextName string) ([]model.ResourceTypeEntry, error) {
	dynClient, err := c.dynamicForContext(contextName)
	if err != nil {
		return nil, err
	}

	crdGVR := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}

	list, err := dynClient.Resource(crdGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing CRDs: %w", err)
	}

	entries := make([]model.ResourceTypeEntry, 0, len(list.Items))
	for _, item := range list.Items {
		spec, ok := item.Object["spec"].(map[string]interface{})
		if !ok {
			continue
		}

		group, _ := spec["group"].(string)

		names, ok := spec["names"].(map[string]interface{})
		if !ok {
			continue
		}
		plural, _ := names["plural"].(string)
		kind, _ := names["kind"].(string)
		if plural == "" || kind == "" {
			continue
		}

		// Determine the preferred/served version.
		apiVersion := preferredCRDVersion(spec, item.Object)

		// Determine scope.
		scope, _ := spec["scope"].(string)
		namespaced := strings.EqualFold(scope, "Namespaced")

		// Build a display name from the plural name (title case the first letter).
		displayName := strings.ToUpper(plural[:1]) + plural[1:]

		entry := model.ResourceTypeEntry{
			DisplayName:    displayName,
			Kind:           kind,
			APIGroup:       group,
			APIVersion:     apiVersion,
			Resource:       plural,
			Icon:           "⧫",
			Namespaced:     namespaced,
			PrinterColumns: extractCRDPrinterColumns(spec, apiVersion),
		}

		// Check if this CRD uses a deprecated API version.
		if dep, found := CheckDeprecation(group, apiVersion, plural); found {
			entry.Deprecated = true
			entry.DeprecationMsg = dep.Message
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].APIGroup != entries[j].APIGroup {
			return entries[i].APIGroup < entries[j].APIGroup
		}
		return entries[i].DisplayName < entries[j].DisplayName
	})

	return entries, nil
}

// preferredCRDVersion extracts the preferred or first served version from a CRD object.
func preferredCRDVersion(spec, obj map[string]interface{}) string {
	// Try spec.versions (v1 CRDs): pick the first one marked as "served", preferring "storage".
	if versions, ok := spec["versions"].([]interface{}); ok && len(versions) > 0 {
		var firstServed, storageVersion string
		for _, v := range versions {
			vm, ok := v.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := vm["name"].(string)
			served, _ := vm["served"].(bool)
			storage, _ := vm["storage"].(bool)
			if storage && served && name != "" {
				storageVersion = name
			}
			if served && firstServed == "" && name != "" {
				firstServed = name
			}
		}
		if storageVersion != "" {
			return storageVersion
		}
		if firstServed != "" {
			return firstServed
		}
	}

	// Fallback: status.storedVersions.
	if status, ok := obj["status"].(map[string]interface{}); ok {
		if stored, ok := status["storedVersions"].([]interface{}); ok && len(stored) > 0 {
			if v, ok := stored[0].(string); ok && v != "" {
				return v
			}
		}
	}

	return "v1"
}

// extractCRDPrinterColumns extracts additionalPrinterColumns from the CRD spec
// for the given preferred version. It skips the "Age" column since age is already
// computed by the application.
func extractCRDPrinterColumns(spec map[string]interface{}, preferredVersion string) []model.PrinterColumn {
	versions, ok := spec["versions"].([]interface{})
	if !ok {
		return nil
	}

	// Find the version entry matching the preferred version.
	for _, v := range versions {
		vm, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := vm["name"].(string)
		if name != preferredVersion {
			continue
		}

		cols, ok := vm["additionalPrinterColumns"].([]interface{})
		if !ok || len(cols) == 0 {
			return nil
		}

		var result []model.PrinterColumn
		for _, c := range cols {
			cm, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			colName, _ := cm["name"].(string)
			colType, _ := cm["type"].(string)
			jsonPath, _ := cm["jsonPath"].(string)
			if colName == "" || jsonPath == "" {
				continue
			}
			// Skip "Age" — the app computes it from creationTimestamp.
			if strings.EqualFold(colName, "Age") {
				continue
			}
			result = append(result, model.PrinterColumn{
				Name:     colName,
				Type:     colType,
				JSONPath: jsonPath,
			})
		}
		return result
	}

	return nil
}

// buildContainerItem creates a model.Item for a container with enriched details.
func buildContainerItem(c corev1.Container, statuses []corev1.ContainerStatus, isInit, isSidecar bool) model.Item {
	item := model.Item{
		Name:  c.Name,
		Kind:  "Container",
		Extra: c.Image,
	}

	switch {
	case isSidecar:
		item.Category = "Sidecar Containers"
		item.Status = "Waiting"
	case isInit:
		item.Category = "Init Containers"
		item.Status = "Init"
	default:
		item.Category = "Containers"
		item.Status = "Waiting"
	}

	// Find matching container status.
	for _, cs := range statuses {
		if cs.Name != c.Name {
			continue
		}

		item.Status = containerStateString(cs.Ready, cs.State.Waiting, cs.State.Running, cs.State.Terminated)
		item.Ready = fmt.Sprintf("%v", cs.Ready)
		item.Restarts = fmt.Sprintf("%d", cs.RestartCount)

		// Started time for age calculation.
		if cs.State.Running != nil && !cs.State.Running.StartedAt.IsZero() {
			item.CreatedAt = cs.State.Running.StartedAt.Time
			item.Age = formatAge(time.Since(cs.State.Running.StartedAt.Time))
		}

		// Add reason to columns if not ready.
		if !cs.Ready {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				item.Status = cs.State.Waiting.Reason
				item.Columns = append(item.Columns, model.KeyValue{Key: "Reason", Value: cs.State.Waiting.Reason})
				if cs.State.Waiting.Message != "" {
					item.Columns = append(item.Columns, model.KeyValue{Key: "Message", Value: cs.State.Waiting.Message})
				}
			}
			if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
				item.Status = cs.State.Terminated.Reason
				item.Columns = append(item.Columns, model.KeyValue{Key: "Reason", Value: cs.State.Terminated.Reason})
				if cs.State.Terminated.Message != "" {
					item.Columns = append(item.Columns, model.KeyValue{Key: "Message", Value: cs.State.Terminated.Message})
				}
				item.Columns = append(item.Columns, model.KeyValue{Key: "Exit Code", Value: fmt.Sprintf("%d", cs.State.Terminated.ExitCode)})
			}
		}

		// Last terminated state (useful for CrashLoopBackOff).
		if cs.LastTerminationState.Terminated != nil {
			lt := cs.LastTerminationState.Terminated
			item.Columns = append(item.Columns, model.KeyValue{Key: "Last Terminated", Value: lt.Reason})
			if lt.ExitCode != 0 {
				item.Columns = append(item.Columns, model.KeyValue{Key: "Last Exit Code", Value: fmt.Sprintf("%d", lt.ExitCode)})
			}
		}

		break
	}

	// Resource requests/limits.
	if req := c.Resources.Requests; req != nil {
		if cpu, ok := req[corev1.ResourceCPU]; ok {
			item.Columns = append(item.Columns, model.KeyValue{Key: "CPU Request", Value: cpu.String()})
		}
		if mem, ok := req[corev1.ResourceMemory]; ok {
			item.Columns = append(item.Columns, model.KeyValue{Key: "Memory Request", Value: mem.String()})
		}
	}
	if lim := c.Resources.Limits; lim != nil {
		if cpu, ok := lim[corev1.ResourceCPU]; ok {
			item.Columns = append(item.Columns, model.KeyValue{Key: "CPU Limit", Value: cpu.String()})
		}
		if mem, ok := lim[corev1.ResourceMemory]; ok {
			item.Columns = append(item.Columns, model.KeyValue{Key: "Memory Limit", Value: mem.String()})
		}
	}

	// Ports.
	if len(c.Ports) > 0 {
		ports := make([]string, 0, len(c.Ports))
		for _, p := range c.Ports {
			port := fmt.Sprintf("%d/%s", p.ContainerPort, p.Protocol)
			if p.Name != "" {
				port = p.Name + ":" + port
			}
			ports = append(ports, port)
		}
		item.Columns = append(item.Columns, model.KeyValue{Key: "Ports", Value: strings.Join(ports, ", ")})
	}

	return item
}

func containerStateString(ready bool, waiting *corev1.ContainerStateWaiting, running *corev1.ContainerStateRunning, terminated *corev1.ContainerStateTerminated) string {
	if running != nil {
		if ready {
			return "Running"
		}
		return "NotReady"
	}
	if waiting != nil {
		return "Waiting"
	}
	if terminated != nil {
		if terminated.Reason == "Completed" {
			return "Completed"
		}
		return "Terminated"
	}
	return "Unknown"
}

// extractContainerNotReadyReason extracts the reason why a container is not ready
// from container statuses (e.g., CrashLoopBackOff, ImagePullBackOff, OOMKilled).
func extractContainerNotReadyReason(containerStatuses []interface{}) string {
	for _, cs := range containerStatuses {
		csMap, ok := cs.(map[string]interface{})
		if !ok {
			continue
		}
		if ready, ok := csMap["ready"].(bool); ok && ready {
			continue
		}
		state, _ := csMap["state"].(map[string]interface{})
		if state == nil {
			continue
		}
		// Check waiting state.
		if waiting, ok := state["waiting"].(map[string]interface{}); ok {
			if reason, ok := waiting["reason"].(string); ok && reason != "" {
				return reason
			}
		}
		// Check terminated state.
		if terminated, ok := state["terminated"].(map[string]interface{}); ok {
			if reason, ok := terminated["reason"].(string); ok && reason != "" {
				return reason
			}
		}
	}
	return ""
}

// extractStatus pulls a human-readable status string from an unstructured object.
func extractStatus(obj map[string]interface{}) string {
	status, ok := obj["status"]
	if !ok {
		return ""
	}
	statusMap, ok := status.(map[string]interface{})
	if !ok {
		return ""
	}
	if phase, ok := statusMap["phase"].(string); ok {
		return phase
	}
	// ArgoCD Application: prefer health status + sync status.
	if health, ok := statusMap["health"].(map[string]interface{}); ok {
		if healthStatus, ok := health["status"].(string); ok {
			if sync, ok := statusMap["sync"].(map[string]interface{}); ok {
				if syncStatus, ok := sync["status"].(string); ok {
					return healthStatus + "/" + syncStatus
				}
			}
			return healthStatus
		}
	}
	// FluxCD resources: check suspend and Ready condition.
	if spec, ok := obj["spec"].(map[string]interface{}); ok {
		if suspended, ok := spec["suspend"].(bool); ok && suspended {
			return "Suspended"
		}
	}
	if conditions, ok := statusMap["conditions"].([]interface{}); ok && len(conditions) > 0 {
		// Prefer "Available" condition with status "True" over other conditions
		// (e.g., Deployments often have "Progressing" as the last condition even
		// when fully available).
		for _, c := range conditions {
			cond, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			condType, _ := cond["type"].(string)
			condStatus, _ := cond["status"].(string)
			if condType == "Available" && condStatus == "True" {
				return "Available"
			}
		}
		// Fall back to the last condition's type.
		if cond, ok := conditions[len(conditions)-1].(map[string]interface{}); ok {
			if t, ok := cond["type"].(string); ok {
				return t
			}
		}
	}
	return ""
}

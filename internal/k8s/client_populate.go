package k8s

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/janosmiko/lfk/internal/model"
)

// populateResourceDetails fills in Ready and Restarts fields for specific resource kinds.
func populateResourceDetails(ti *model.Item, obj map[string]interface{}, kind string) {
	status, _ := obj["status"].(map[string]interface{})
	spec, _ := obj["spec"].(map[string]interface{})

	switch kind {
	case "Pod":
		if status == nil {
			return
		}
		containerStatuses, _ := status["containerStatuses"].([]interface{})
		totalContainers := len(containerStatuses)
		if containers, ok := spec["containers"].([]interface{}); ok {
			totalContainers = len(containers)
		}
		readyCount := 0
		restartCount := int64(0)
		for _, cs := range containerStatuses {
			csMap, ok := cs.(map[string]interface{})
			if !ok {
				continue
			}
			if ready, ok := csMap["ready"].(bool); ok && ready {
				readyCount++
			}
			if rc, ok := csMap["restartCount"].(int64); ok {
				restartCount += rc
			} else if rcf, ok := csMap["restartCount"].(float64); ok {
				restartCount += int64(rcf)
			}
		}
		ti.Ready = fmt.Sprintf("%d/%d", readyCount, totalContainers)
		ti.Restarts = fmt.Sprintf("%d", restartCount)

		// Find the most recent restart time from container lastState.
		var lastRestart time.Time
		for _, cs := range containerStatuses {
			csMap, ok := cs.(map[string]interface{})
			if !ok {
				continue
			}
			lastState, _ := csMap["lastState"].(map[string]interface{})
			if lastState == nil {
				continue
			}
			if terminated, ok := lastState["terminated"].(map[string]interface{}); ok {
				if finishedAt, ok := terminated["finishedAt"].(string); ok {
					if t, err := time.Parse(time.RFC3339, finishedAt); err == nil {
						if t.After(lastRestart) {
							lastRestart = t
						}
					}
				}
			}
		}
		ti.LastRestartAt = lastRestart

		// Override status based on container readiness.
		// Succeeded pods stay green even with unready containers.
		if ti.Status != "Succeeded" && readyCount < totalContainers && totalContainers > 0 {
			// Check init container statuses first — when an init container fails,
			// regular containers show "PodInitializing" which hides the real reason.
			initContainerStatuses, _ := status["initContainerStatuses"].([]interface{})
			reason := extractContainerNotReadyReason(initContainerStatuses)
			if reason == "" || reason == "PodInitializing" {
				reason = extractContainerNotReadyReason(containerStatuses)
			}
			// If the pod phase is Failed, prefer that over "PodInitializing".
			if reason == "PodInitializing" && ti.Status == "Failed" {
				reason = ""
			}
			if reason != "" {
				ti.Status = reason
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Reason", Value: reason})
			} else if ti.Status == "Running" {
				ti.Status = "NotReady"
			}
		}

		// Resource requests/limits from container specs.
		if containers, ok := spec["containers"].([]interface{}); ok {
			cpuReq, cpuLim, memReq, memLim := extractContainerResources(containers)
			addResourceColumns(ti, cpuReq, cpuLim, memReq, memLim)
		}

		// Additional columns for preview.
		if qos, ok := status["qosClass"].(string); ok {
			ti.Columns = append(ti.Columns, model.KeyValue{Key: "QoS", Value: qos})
		}
		if sa, ok := spec["serviceAccountName"].(string); ok {
			ti.Columns = append(ti.Columns, model.KeyValue{Key: "Service Account", Value: sa})
		}
		if podIP, ok := status["podIP"].(string); ok {
			ti.Columns = append(ti.Columns, model.KeyValue{Key: "Pod IP", Value: podIP})
		}
		if containers, ok := spec["containers"].([]interface{}); ok {
			var images []string
			for _, c := range containers {
				if cMap, ok := c.(map[string]interface{}); ok {
					if img, ok := cMap["image"].(string); ok {
						images = append(images, img)
					}
				}
			}
			if len(images) > 0 {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Images", Value: strings.Join(images, ", ")})
			}
		}
		// Priority class.
		if pc, ok := spec["priorityClassName"].(string); ok && pc != "" {
			ti.Columns = append(ti.Columns, model.KeyValue{Key: "Priority Class", Value: pc})
		}
		// Node at the end (lower priority in table view).
		if nodeName, ok := spec["nodeName"].(string); ok {
			ti.Columns = append(ti.Columns, model.KeyValue{Key: "Node", Value: nodeName})
		}

	case "Deployment":
		if status == nil || spec == nil {
			return
		}
		var specReplicas int64 = 1
		if r, ok := spec["replicas"].(int64); ok {
			specReplicas = r
		} else if r, ok := spec["replicas"].(float64); ok {
			specReplicas = int64(r)
		}
		var readyReplicas int64
		if r, ok := status["readyReplicas"].(int64); ok {
			readyReplicas = r
		} else if r, ok := status["readyReplicas"].(float64); ok {
			readyReplicas = int64(r)
		}
		ti.Ready = fmt.Sprintf("%d/%d", readyReplicas, specReplicas)
		// Additional columns.
		ti.Columns = append(ti.Columns, model.KeyValue{Key: "Replicas", Value: fmt.Sprintf("%d", specReplicas)})
		if strategy, ok := spec["strategy"].(map[string]interface{}); ok {
			if t, ok := strategy["type"].(string); ok {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Strategy", Value: t})
			}
		}
		if updated, ok := status["updatedReplicas"].(float64); ok {
			ti.Columns = append(ti.Columns, model.KeyValue{Key: "Up-to-date", Value: fmt.Sprintf("%d", int64(updated))})
		}
		if avail, ok := status["availableReplicas"].(float64); ok {
			ti.Columns = append(ti.Columns, model.KeyValue{Key: "Available", Value: fmt.Sprintf("%d", int64(avail))})
		}
		// Aggregated resource requests/limits (per-pod from template).
		cpuReq, cpuLim, memReq, memLim := extractTemplateResources(spec)
		addResourceColumns(ti, cpuReq, cpuLim, memReq, memLim)
		populateContainerImages(ti, spec)

	case "StatefulSet":
		if status == nil || spec == nil {
			return
		}
		var specReplicas int64 = 1
		if r, ok := spec["replicas"].(int64); ok {
			specReplicas = r
		} else if r, ok := spec["replicas"].(float64); ok {
			specReplicas = int64(r)
		}
		var readyReplicas int64
		if r, ok := status["readyReplicas"].(int64); ok {
			readyReplicas = r
		} else if r, ok := status["readyReplicas"].(float64); ok {
			readyReplicas = int64(r)
		}
		ti.Ready = fmt.Sprintf("%d/%d", readyReplicas, specReplicas)
		ti.Columns = append(ti.Columns, model.KeyValue{Key: "Replicas", Value: fmt.Sprintf("%d", specReplicas)})
		// Aggregated resource requests/limits (per-pod from template).
		cpuReq, cpuLim, memReq, memLim := extractTemplateResources(spec)
		addResourceColumns(ti, cpuReq, cpuLim, memReq, memLim)
		populateContainerImages(ti, spec)

	case "DaemonSet":
		if status == nil {
			return
		}
		var desired, ready int64
		if d, ok := status["desiredNumberScheduled"].(int64); ok {
			desired = d
		} else if d, ok := status["desiredNumberScheduled"].(float64); ok {
			desired = int64(d)
		}
		if r, ok := status["numberReady"].(int64); ok {
			ready = r
		} else if r, ok := status["numberReady"].(float64); ok {
			ready = int64(r)
		}
		ti.Ready = fmt.Sprintf("%d/%d", ready, desired)
		ti.Columns = append(ti.Columns, model.KeyValue{Key: "Desired", Value: fmt.Sprintf("%d", desired)})
		// Per-pod resource requests/limits from template.
		if spec != nil {
			cpuReq, cpuLim, memReq, memLim := extractTemplateResources(spec)
			addResourceColumns(ti, cpuReq, cpuLim, memReq, memLim)
		}

	case "ReplicaSet":
		if status == nil || spec == nil {
			return
		}
		var specReplicas int64
		if r, ok := spec["replicas"].(int64); ok {
			specReplicas = r
		} else if r, ok := spec["replicas"].(float64); ok {
			specReplicas = int64(r)
		}
		var readyReplicas int64
		if r, ok := status["readyReplicas"].(int64); ok {
			readyReplicas = r
		} else if r, ok := status["readyReplicas"].(float64); ok {
			readyReplicas = int64(r)
		}
		ti.Ready = fmt.Sprintf("%d/%d", readyReplicas, specReplicas)

	case "Service":
		if spec == nil {
			return
		}
		if svcType, ok := spec["type"].(string); ok {
			ti.Columns = append(ti.Columns, model.KeyValue{Key: "Type", Value: svcType})
		}
		if clusterIP, ok := spec["clusterIP"].(string); ok {
			ti.Columns = append(ti.Columns, model.KeyValue{Key: "Cluster IP", Value: clusterIP})
		}
		if ports, ok := spec["ports"].([]interface{}); ok {
			var portStrs []string
			for _, p := range ports {
				if pMap, ok := p.(map[string]interface{}); ok {
					port := getInt(pMap, "port")
					proto, _ := pMap["protocol"].(string)
					s := fmt.Sprintf("%d/%s", port, proto)
					if tp := getInt(pMap, "targetPort"); tp > 0 && tp != port {
						s = fmt.Sprintf("%d→%d/%s", port, tp, proto)
					}
					portStrs = append(portStrs, s)
				}
			}
			if len(portStrs) > 0 {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Ports", Value: strings.Join(portStrs, ", ")})
			}
		}
		// External IPs from spec.
		if extIPs, ok := spec["externalIPs"].([]interface{}); ok && len(extIPs) > 0 {
			var ips []string
			for _, ip := range extIPs {
				if s, ok := ip.(string); ok {
					ips = append(ips, s)
				}
			}
			if len(ips) > 0 {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "External IPs", Value: strings.Join(ips, ", ")})
			}
		}
		// External IP from LoadBalancer status.
		if status != nil {
			if lb, ok := status["loadBalancer"].(map[string]interface{}); ok {
				if ingress, ok := lb["ingress"].([]interface{}); ok {
					var addrs []string
					for _, i := range ingress {
						if iMap, ok := i.(map[string]interface{}); ok {
							if ip, ok := iMap["ip"].(string); ok {
								addrs = append(addrs, ip)
							} else if host, ok := iMap["hostname"].(string); ok {
								addrs = append(addrs, host)
							}
						}
					}
					if len(addrs) > 0 {
						ti.Columns = append(ti.Columns, model.KeyValue{Key: "External Address", Value: strings.Join(addrs, ", ")})
					}
				}
			}
		}
		if selector, ok := spec["selector"].(map[string]interface{}); ok {
			var parts []string
			for k, v := range selector {
				parts = append(parts, fmt.Sprintf("%s=%v", k, v))
			}
			sort.Strings(parts)
			if len(parts) > 0 {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Selector", Value: strings.Join(parts, ", ")})
			}
		}
		if spec["sessionAffinity"] != nil {
			if sa, ok := spec["sessionAffinity"].(string); ok && sa != "None" {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Session Affinity", Value: sa})
			}
		}

	case "Ingress":
		if spec == nil {
			return
		}
		// Ingress class.
		if ic, ok := spec["ingressClassName"].(string); ok && ic != "" {
			ti.Columns = append(ti.Columns, model.KeyValue{Key: "Ingress Class", Value: ic})
		}
		if rules, ok := spec["rules"].([]interface{}); ok {
			ti.Columns = append(ti.Columns, model.KeyValue{Key: "Rules", Value: fmt.Sprintf("%d", len(rules))})
			var hosts []string
			for _, r := range rules {
				if rMap, ok := r.(map[string]interface{}); ok {
					if host, ok := rMap["host"].(string); ok {
						hosts = append(hosts, host)
					}
				}
			}
			if len(hosts) > 0 {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Hosts", Value: strings.Join(hosts, ", ")})
			}
		}
		// Default backend.
		if defBackend, ok := spec["defaultBackend"].(map[string]interface{}); ok {
			if svc, ok := defBackend["service"].(map[string]interface{}); ok {
				svcName, _ := svc["name"].(string)
				if port, ok := svc["port"].(map[string]interface{}); ok {
					if num, ok := port["number"].(float64); ok {
						ti.Columns = append(ti.Columns, model.KeyValue{Key: "Default Backend", Value: fmt.Sprintf("%s:%d", svcName, int64(num))})
					} else if name, ok := port["name"].(string); ok {
						ti.Columns = append(ti.Columns, model.KeyValue{Key: "Default Backend", Value: fmt.Sprintf("%s:%s", svcName, name)})
					}
				} else if svcName != "" {
					ti.Columns = append(ti.Columns, model.KeyValue{Key: "Default Backend", Value: svcName})
				}
			}
		}
		// TLS hosts.
		var tlsHostSet map[string]bool
		if tls, ok := spec["tls"].([]interface{}); ok && len(tls) > 0 {
			tlsHostSet = make(map[string]bool)
			var tlsHosts []string
			for _, t := range tls {
				if tMap, ok := t.(map[string]interface{}); ok {
					if hosts, ok := tMap["hosts"].([]interface{}); ok {
						for _, h := range hosts {
							if s, ok := h.(string); ok {
								tlsHosts = append(tlsHosts, s)
								tlsHostSet[s] = true
							}
						}
					}
				}
			}
			if len(tlsHosts) > 0 {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "TLS Hosts", Value: strings.Join(tlsHosts, ", ")})
			}
		}
		// Build a URL from the first rule's host and path for "Open in Browser".
		if rules, ok := spec["rules"].([]interface{}); ok && len(rules) > 0 {
			if firstRule, ok := rules[0].(map[string]interface{}); ok {
				if host, ok := firstRule["host"].(string); ok && host != "" {
					scheme := "http"
					if tlsHostSet[host] {
						scheme = "https"
					}
					path := ""
					if httpBlock, ok := firstRule["http"].(map[string]interface{}); ok {
						if paths, ok := httpBlock["paths"].([]interface{}); ok && len(paths) > 0 {
							if firstPath, ok := paths[0].(map[string]interface{}); ok {
								if p, ok := firstPath["path"].(string); ok && p != "" && p != "/" {
									path = p
								}
							}
						}
					}
					ti.Columns = append(ti.Columns, model.KeyValue{Key: "__ingress_url", Value: scheme + "://" + host + path})
				}
			}
		}
		if status != nil {
			if lb, ok := status["loadBalancer"].(map[string]interface{}); ok {
				if ingress, ok := lb["ingress"].([]interface{}); ok {
					var addrs []string
					for _, i := range ingress {
						if iMap, ok := i.(map[string]interface{}); ok {
							if ip, ok := iMap["ip"].(string); ok {
								addrs = append(addrs, ip)
							} else if host, ok := iMap["hostname"].(string); ok {
								addrs = append(addrs, host)
							}
						}
					}
					if len(addrs) > 0 {
						ti.Columns = append(ti.Columns, model.KeyValue{Key: "Address", Value: strings.Join(addrs, ", ")})
					}
				}
			}
		}

	case "ConfigMap":
		if data, ok := obj["data"].(map[string]interface{}); ok {
			var keys []string
			for k := range data {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			// Store ConfigMap data values with "data:" prefix for preview display.
			for _, k := range keys {
				if v, ok := data[k].(string); ok {
					ti.Columns = append(ti.Columns, model.KeyValue{Key: "data:" + k, Value: v})
				}
			}
		}

	case "Secret":
		if data, ok := obj["data"].(map[string]interface{}); ok {
			var keys []string
			for k := range data {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			// Store decoded secret values with "secret:" prefix for conditional display.
			for _, k := range keys {
				if encoded, ok := data[k].(string); ok {
					decoded, err := base64.StdEncoding.DecodeString(encoded)
					if err == nil {
						ti.Columns = append(ti.Columns, model.KeyValue{Key: "secret:" + k, Value: string(decoded)})
					}
				}
			}
		}
		if sType, ok := obj["type"].(string); ok {
			ti.Columns = append(ti.Columns, model.KeyValue{Key: "Type", Value: sType})
		}

	case "Node":
		// Extract role from labels.
		if metadata, ok := obj["metadata"].(map[string]interface{}); ok {
			if labels, ok := metadata["labels"].(map[string]interface{}); ok {
				var roles []string
				for k := range labels {
					if strings.HasPrefix(k, "node-role.kubernetes.io/") {
						role := strings.TrimPrefix(k, "node-role.kubernetes.io/")
						if role != "" {
							roles = append(roles, role)
						}
					}
				}
				if len(roles) > 0 {
					sort.Strings(roles)
					ti.Columns = append(ti.Columns, model.KeyValue{Key: "Role", Value: strings.Join(roles, ",")})
				}
			}
		}

		if status != nil {
			if addrs, ok := status["addresses"].([]interface{}); ok {
				for _, a := range addrs {
					if aMap, ok := a.(map[string]interface{}); ok {
						addrType, _ := aMap["type"].(string)
						addr, _ := aMap["address"].(string)
						if addrType != "" && addr != "" {
							ti.Columns = append(ti.Columns, model.KeyValue{Key: addrType, Value: addr})
						}
					}
				}
			}
			// Add allocatable CPU/Memory as hidden data columns for metrics enrichment.
			if alloc, ok := status["allocatable"].(map[string]interface{}); ok {
				if cpu, ok := alloc["cpu"].(string); ok {
					ti.Columns = append(ti.Columns, model.KeyValue{Key: "CPU Alloc", Value: cpu})
				}
				if mem, ok := alloc["memory"].(string); ok {
					ti.Columns = append(ti.Columns, model.KeyValue{Key: "Mem Alloc", Value: mem})
				}
			}
			if nodeInfo, ok := status["nodeInfo"].(map[string]interface{}); ok {
				if v, ok := nodeInfo["kubeletVersion"].(string); ok {
					ti.Columns = append(ti.Columns, model.KeyValue{Key: "Version", Value: v})
				}
				if v, ok := nodeInfo["osImage"].(string); ok {
					ti.Columns = append(ti.Columns, model.KeyValue{Key: "OS", Value: v})
				}
				if v, ok := nodeInfo["containerRuntimeVersion"].(string); ok {
					ti.Columns = append(ti.Columns, model.KeyValue{Key: "Runtime", Value: v})
				}
			}
		}

		// Extract taints from spec.
		if spec != nil {
			if taints, ok := spec["taints"].([]interface{}); ok && len(taints) > 0 {
				var taintStrs []string
				for _, t := range taints {
					if tMap, ok := t.(map[string]interface{}); ok {
						key, _ := tMap["key"].(string)
						value, _ := tMap["value"].(string)
						effect, _ := tMap["effect"].(string)
						taint := key
						if value != "" {
							taint += "=" + value
						}
						taint += ":" + effect
						taintStrs = append(taintStrs, taint)
					}
				}
				if len(taintStrs) > 0 {
					ti.Columns = append(ti.Columns, model.KeyValue{Key: "Taints", Value: strings.Join(taintStrs, ", ")})
				}
			}
		}

	case "PersistentVolumeClaim":
		// Phase/status.
		if status != nil {
			if phase, ok := status["phase"].(string); ok {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Status", Value: phase})
				ti.Status = phase
			}
			// Actual capacity from status (may differ from requested).
			if cap, ok := status["capacity"].(map[string]interface{}); ok {
				if storage, ok := cap["storage"].(string); ok {
					ti.Columns = append(ti.Columns, model.KeyValue{Key: "Capacity", Value: storage})
				}
			}
		}
		if spec != nil {
			// Requested storage (show if no status capacity yet).
			if res, ok := spec["resources"].(map[string]interface{}); ok {
				if req, ok := res["requests"].(map[string]interface{}); ok {
					if storage, ok := req["storage"].(string); ok {
						ti.Columns = append(ti.Columns, model.KeyValue{Key: "Request", Value: storage})
					}
				}
			}
			// Volume name.
			if vol, ok := spec["volumeName"].(string); ok && vol != "" {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Volume", Value: vol})
			}
			if am, ok := spec["accessModes"].([]interface{}); ok {
				var modes []string
				for _, m := range am {
					if s, ok := m.(string); ok {
						modes = append(modes, s)
					}
				}
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Access Modes", Value: strings.Join(modes, ", ")})
			}
			if sc, ok := spec["storageClassName"].(string); ok {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Storage Class", Value: sc})
			}
			if vm, ok := spec["volumeMode"].(string); ok && vm != "" {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Volume Mode", Value: vm})
			}
		}

	case "CronJob":
		if spec != nil {
			if sched, ok := spec["schedule"].(string); ok {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Schedule", Value: sched})
			}
			if suspend, ok := spec["suspend"].(bool); ok {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Suspend", Value: fmt.Sprintf("%v", suspend)})
			}
		}
		if status != nil {
			if lastSchedule, ok := status["lastScheduleTime"].(string); ok {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Last Schedule", Value: lastSchedule})
			}
		}

	case "Job":
		if status != nil {
			if succeeded, ok := status["succeeded"].(float64); ok {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Succeeded", Value: fmt.Sprintf("%d", int64(succeeded))})
			}
			if failed, ok := status["failed"].(float64); ok && failed > 0 {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Failed", Value: fmt.Sprintf("%d", int64(failed))})
			}
		}
		if spec != nil {
			if completions, ok := spec["completions"].(float64); ok {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Completions", Value: fmt.Sprintf("%d", int64(completions))})
			}
		}

	case "HorizontalPodAutoscaler":
		// Set Ready field to show current/desired replicas in the list table.
		if status != nil {
			var currentR, desiredR int64
			if cr, ok := status["currentReplicas"].(float64); ok {
				currentR = int64(cr)
			}
			if dr, ok := status["desiredReplicas"].(float64); ok {
				desiredR = int64(dr)
			}
			// Show min/max from spec for context.
			var minR, maxR int64
			if spec != nil {
				if mr, ok := spec["minReplicas"].(float64); ok {
					minR = int64(mr)
				}
				if mr, ok := spec["maxReplicas"].(float64); ok {
					maxR = int64(mr)
				}
			}
			ti.Ready = fmt.Sprintf("%d/%d (%d-%d)", currentR, desiredR, minR, maxR)
		}
		if spec != nil {
			// Target reference.
			if scaleTargetRef, ok := spec["scaleTargetRef"].(map[string]interface{}); ok {
				refKind, _ := scaleTargetRef["kind"].(string)
				refName, _ := scaleTargetRef["name"].(string)
				if refKind != "" && refName != "" {
					ti.Columns = append(ti.Columns, model.KeyValue{Key: "Target", Value: refKind + "/" + refName})
				}
			}
			if minR, ok := spec["minReplicas"].(float64); ok {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Min Replicas", Value: fmt.Sprintf("%d", int64(minR))})
			}
			if maxR, ok := spec["maxReplicas"].(float64); ok {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Max Replicas", Value: fmt.Sprintf("%d", int64(maxR))})
			}
			// Metrics from spec (target values).
			if metrics, ok := spec["metrics"].([]interface{}); ok {
				for _, m := range metrics {
					mMap, ok := m.(map[string]interface{})
					if !ok {
						continue
					}
					mType, _ := mMap["type"].(string)
					switch mType {
					case "Resource":
						if res, ok := mMap["resource"].(map[string]interface{}); ok {
							resName, _ := res["name"].(string)
							if target, ok := res["target"].(map[string]interface{}); ok {
								targetType, _ := target["type"].(string)
								switch targetType {
								case "Utilization":
									if avg, ok := target["averageUtilization"].(float64); ok {
										ti.Columns = append(ti.Columns, model.KeyValue{
											Key:   fmt.Sprintf("Target %s", strings.ToUpper(resName[:1])+resName[1:]),
											Value: fmt.Sprintf("%d%%", int64(avg)),
										})
									}
								case "AverageValue":
									if avg, ok := target["averageValue"].(string); ok {
										ti.Columns = append(ti.Columns, model.KeyValue{
											Key:   fmt.Sprintf("Target %s", strings.ToUpper(resName[:1])+resName[1:]),
											Value: avg,
										})
									}
								}
							}
						}
					case "Pods":
						if pods, ok := mMap["pods"].(map[string]interface{}); ok {
							metricName := ""
							if mn, ok := pods["metric"].(map[string]interface{}); ok {
								metricName, _ = mn["name"].(string)
							}
							if target, ok := pods["target"].(map[string]interface{}); ok {
								if avg, ok := target["averageValue"].(string); ok && metricName != "" {
									ti.Columns = append(ti.Columns, model.KeyValue{
										Key:   fmt.Sprintf("Target %s", metricName),
										Value: avg,
									})
								}
							}
						}
					case "Object":
						if object, ok := mMap["object"].(map[string]interface{}); ok {
							metricName := ""
							if mn, ok := object["metric"].(map[string]interface{}); ok {
								metricName, _ = mn["name"].(string)
							}
							if target, ok := object["target"].(map[string]interface{}); ok {
								if val, ok := target["value"].(string); ok && metricName != "" {
									ti.Columns = append(ti.Columns, model.KeyValue{
										Key:   fmt.Sprintf("Target %s", metricName),
										Value: val,
									})
								}
							}
						}
					}
				}
			}
		}
		if status != nil {
			if current, ok := status["currentReplicas"].(float64); ok {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Current Replicas", Value: fmt.Sprintf("%d", int64(current))})
			}
			if desired, ok := status["desiredReplicas"].(float64); ok {
				ti.Columns = append(ti.Columns, model.KeyValue{Key: "Desired Replicas", Value: fmt.Sprintf("%d", int64(desired))})
			}
			// Current metrics from status.
			if currentMetrics, ok := status["currentMetrics"].([]interface{}); ok {
				for _, m := range currentMetrics {
					mMap, ok := m.(map[string]interface{})
					if !ok {
						continue
					}
					mType, _ := mMap["type"].(string)
					switch mType {
					case "Resource":
						if res, ok := mMap["resource"].(map[string]interface{}); ok {
							resName, _ := res["name"].(string)
							if current, ok := res["current"].(map[string]interface{}); ok {
								if avg, ok := current["averageUtilization"].(float64); ok {
									ti.Columns = append(ti.Columns, model.KeyValue{
										Key:   fmt.Sprintf("Current %s", strings.ToUpper(resName[:1])+resName[1:]),
										Value: fmt.Sprintf("%d%%", int64(avg)),
									})
								} else if avgVal, ok := current["averageValue"].(string); ok {
									ti.Columns = append(ti.Columns, model.KeyValue{
										Key:   fmt.Sprintf("Current %s", strings.ToUpper(resName[:1])+resName[1:]),
										Value: avgVal,
									})
								}
							}
						}
					case "Pods":
						if pods, ok := mMap["pods"].(map[string]interface{}); ok {
							metricName := ""
							if mn, ok := pods["metric"].(map[string]interface{}); ok {
								metricName, _ = mn["name"].(string)
							}
							if current, ok := pods["current"].(map[string]interface{}); ok {
								if avg, ok := current["averageValue"].(string); ok && metricName != "" {
									ti.Columns = append(ti.Columns, model.KeyValue{
										Key:   fmt.Sprintf("Current %s", metricName),
										Value: avg,
									})
								}
							}
						}
					}
				}
			}
			// Conditions summary.
			if conditions, ok := status["conditions"].([]interface{}); ok {
				for _, c := range conditions {
					cMap, ok := c.(map[string]interface{})
					if !ok {
						continue
					}
					cType, _ := cMap["type"].(string)
					cStatus, _ := cMap["status"].(string)
					if cType == "ScalingLimited" && cStatus == "True" {
						msg, _ := cMap["message"].(string)
						if msg != "" {
							ti.Columns = append(ti.Columns, model.KeyValue{Key: "Scaling Limited", Value: msg})
						}
					}
				}
			}
		}

	default:
		// Extended kinds (FluxCD, cert-manager, ArgoCD, Events, storage types, etc.)
		// and unknown/CRD resources are handled in a separate file.
		populateResourceDetailsExt(ti, obj, kind, status, spec)
	}
}

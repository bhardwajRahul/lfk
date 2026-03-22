package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// --- populateResourceDetails: Pod with PodInitializing + Failed status ---

func TestPopulateResourceDetails_Pod_InitializingFailedStatus(t *testing.T) {
	// When init container reason is "PodInitializing" and pod status is "Failed",
	// the reason should be cleared (line 81-83).
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{"name": "app"},
			},
		},
		"status": map[string]interface{}{
			"phase": "Failed",
			"initContainerStatuses": []interface{}{
				map[string]interface{}{
					"ready": false,
					"state": map[string]interface{}{
						"waiting": map[string]interface{}{
							"reason": "PodInitializing",
						},
					},
				},
			},
			"containerStatuses": []interface{}{
				map[string]interface{}{
					"name":         "app",
					"ready":        false,
					"restartCount": float64(0),
					"state": map[string]interface{}{
						"waiting": map[string]interface{}{
							"reason": "PodInitializing",
						},
					},
				},
			},
		},
	}

	ti := &model.Item{Status: "Failed"}
	populateResourceDetails(ti, obj, "Pod")

	// With PodInitializing + Failed, reason gets cleared and status stays "Failed".
	assert.Equal(t, "Failed", ti.Status)
}

// --- populateResourceDetails: Ingress with service name only (no port map) ---

func TestPopulateResourceDetails_Ingress_DefaultBackendServiceNameOnly(t *testing.T) {
	// When a default backend has a service with name but no port map (line 336-338).
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"defaultBackend": map[string]interface{}{
				"service": map[string]interface{}{
					"name": "my-backend",
				},
			},
		},
	}

	ti := &model.Item{}
	populateResourceDetails(ti, obj, "Ingress")

	colMap := columnsToMap(ti.Columns)
	assert.Equal(t, "my-backend", colMap["Default Backend"])
}

func TestPopulateResourceDetails_Ingress_DefaultBackendPortName(t *testing.T) {
	// When a default backend has a service with a named port instead of numeric (line 333-334).
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"defaultBackend": map[string]interface{}{
				"service": map[string]interface{}{
					"name": "my-backend",
					"port": map[string]interface{}{
						"name": "https",
					},
				},
			},
		},
	}

	ti := &model.Item{}
	populateResourceDetails(ti, obj, "Ingress")

	colMap := columnsToMap(ti.Columns)
	assert.Equal(t, "my-backend:https", colMap["Default Backend"])
}

func TestPopulateResourceDetails_Ingress_LoadBalancerHostname(t *testing.T) {
	// When a load balancer ingress entry has hostname instead of IP (line 392-394).
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"rules": []interface{}{
				map[string]interface{}{
					"host": "example.com",
				},
			},
		},
		"status": map[string]interface{}{
			"loadBalancer": map[string]interface{}{
				"ingress": []interface{}{
					map[string]interface{}{
						"hostname": "lb.example.com",
					},
				},
			},
		},
	}

	ti := &model.Item{}
	populateResourceDetails(ti, obj, "Ingress")

	colMap := columnsToMap(ti.Columns)
	assert.Equal(t, "lb.example.com", colMap["Address"])
}

// --- populateResourceDetails: HPA with non-map metric entries ---

func TestPopulateResourceDetails_HPA_NonMapSpecMetric(t *testing.T) {
	// Non-map metric entries in spec.metrics should be skipped (line 632-633).
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"maxReplicas": float64(5),
			"metrics": []interface{}{
				"not-a-map",
				map[string]interface{}{
					"type": "Resource",
					"resource": map[string]interface{}{
						"name": "cpu",
						"target": map[string]interface{}{
							"type":               "Utilization",
							"averageUtilization": float64(70),
						},
					},
				},
			},
		},
		"status": map[string]interface{}{
			"currentReplicas": float64(1),
			"desiredReplicas": float64(1),
		},
	}

	ti := &model.Item{}
	populateResourceDetails(ti, obj, "HorizontalPodAutoscaler")

	colMap := columnsToMap(ti.Columns)
	assert.Equal(t, "70%", colMap["Target Cpu"])
}

func TestPopulateResourceDetails_HPA_NonMapCurrentMetric(t *testing.T) {
	// Non-map metric entries in status.currentMetrics should be skipped (line 705-706).
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"maxReplicas": float64(5),
		},
		"status": map[string]interface{}{
			"currentReplicas": float64(1),
			"desiredReplicas": float64(1),
			"currentMetrics": []interface{}{
				"not-a-map",
				map[string]interface{}{
					"type": "Resource",
					"resource": map[string]interface{}{
						"name": "cpu",
						"current": map[string]interface{}{
							"averageUtilization": float64(45),
						},
					},
				},
			},
		},
	}

	ti := &model.Item{}
	populateResourceDetails(ti, obj, "HorizontalPodAutoscaler")

	colMap := columnsToMap(ti.Columns)
	assert.Equal(t, "45%", colMap["Current Cpu"])
}

func TestPopulateResourceDetails_HPA_NonMapCondition(t *testing.T) {
	// Non-map condition entries in status.conditions should be skipped (line 749-750).
	obj := map[string]interface{}{
		"spec": map[string]interface{}{
			"maxReplicas": float64(3),
		},
		"status": map[string]interface{}{
			"currentReplicas": float64(3),
			"desiredReplicas": float64(3),
			"conditions": []interface{}{
				"not-a-map",
				map[string]interface{}{
					"type":    "ScalingLimited",
					"status":  "True",
					"message": "limited",
				},
			},
		},
	}

	ti := &model.Item{}
	populateResourceDetails(ti, obj, "HorizontalPodAutoscaler")

	colMap := columnsToMap(ti.Columns)
	assert.Equal(t, "limited", colMap["Scaling Limited"])
}

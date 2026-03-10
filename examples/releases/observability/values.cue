// Values provide concrete configuration for the Observability Stack module.
// These satisfy the #config schema defined in module.cue.
// Default configuration: single-instance development/staging setup with
// all components enabled, persistent storage, and config reload sidecar.
package main

// Concrete default values
values: {

	// === Global Prometheus Settings ===
	global: {
		scrapeInterval:     "30s"
		scrapeTimeout:      "10s"
		evaluationInterval: "30s"
	}

	// === Prometheus Server ===
	prometheus: {
		image: {
			repository: "quay.io/prometheus/prometheus"
			tag:        "v3.4.1"
			digest:     ""
		}

		// Keep 15 days of metrics data
		retention: "15d"

		// Single instance for dev/staging
		replicas: 1

		// Persistent TSDB storage
		storage: {
			type: "pvc"
			size: "8Gi"
		}

		// Moderate resource allocation
		resources: {
			requests: {
				cpu:    "250m"
				memory: "512Mi"
			}
			limits: {
				cpu:    "1000m"
				memory: "2Gi"
			}
		}

		// Standard Prometheus port
		port: 9090

		// Enable config-reload sidecar for live ConfigMap updates
		configReload: {
			enabled: true
			image: {
				repository: "quay.io/prometheus-operator/prometheus-config-reloader"
				tag:        "v0.89.0"
				digest:     ""
			}
			port: 8080
		}

		// No ingress by default - use port-forward for development
		// ingress: { ... }  // See values_production.cue

		// Service account for Kubernetes API access (scraping via service discovery)
		serviceAccount: name: "prometheus"

		// Enable lifecycle API for config reload via POST /-/reload
		extraFlags: ["web.enable-lifecycle"]
	}

	// === Alertmanager ===
	alertmanager: {
		enabled: true

		image: {
			repository: "quay.io/prometheus/alertmanager"
			tag:        "v0.28.1"
			digest:     ""
		}

		replicas: 1

		storage: {
			type: "pvc"
			size: "2Gi"
		}

		resources: {
			requests: {
				cpu:    "50m"
				memory: "64Mi"
			}
			limits: {
				cpu:    "200m"
				memory: "256Mi"
			}
		}

		port: 9093

		// Default alert routing configuration
		config: {
			route: {
				receiver:       "default"
				groupBy:        ["alertname", "namespace"]
				groupWait:      "30s"
				groupInterval:  "5m"
				repeatInterval: "4h"
			}
			// Default receiver does nothing - users should configure real receivers
			receivers: [{
				name: "default"
			}]
		}
	}

	// === Node Exporter ===
	nodeExporter: {
		enabled: true

		image: {
			repository: "quay.io/prometheus/node-exporter"
			tag:        "v1.9.1"
			digest:     ""
		}

		port: 9100

		resources: {
			requests: {
				cpu:    "50m"
				memory: "32Mi"
			}
			limits: {
				cpu:    "200m"
				memory: "128Mi"
			}
		}

		hostPaths: {
			proc: "/proc"
			sys:  "/sys"
		}
	}

	// === Kube State Metrics ===
	kubeStateMetrics: {
		enabled: true

		image: {
			repository: "registry.k8s.io/kube-state-metrics/kube-state-metrics"
			tag:        "v2.15.0"
			digest:     ""
		}

		port:          8080
		telemetryPort: 8081

		resources: {
			requests: {
				cpu:    "50m"
				memory: "64Mi"
			}
			limits: {
				cpu:    "200m"
				memory: "256Mi"
			}
		}

		serviceAccount: name: "kube-state-metrics"
	}

	// === Pushgateway ===
	pushgateway: {
		enabled: true

		image: {
			repository: "quay.io/prometheus/pushgateway"
			tag:        "v1.11.0"
			digest:     ""
		}

		port: 9091

		resources: {
			requests: {
				cpu:    "25m"
				memory: "32Mi"
			}
			limits: {
				cpu:    "100m"
				memory: "64Mi"
			}
		}
	}

	// === Security ===
	// Run all components as nobody (65534) - standard for Prometheus ecosystem
	security: {
		runAsUser:  65534
		runAsGroup: 65534
	}
}

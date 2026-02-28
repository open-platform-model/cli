// Minimal values for testing and CI environments.
// Apply with: opm mod build examples/observability-stack -f examples/observability-stack/values_minimal.cue
//
// Differences from default (values.cue):
// - Pushgateway disabled (not needed for testing)
// - All storage uses emptyDir (no PVC provisioner required)
// - Config reload sidecar disabled (not needed for ephemeral environments)
// - Short retention (2d) to minimize resource usage
// - Minimal resource allocations
package main

values: {

	// === Global ===
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

		// Short retention for ephemeral environments
		retention: "2d"
		replicas:  1

		// No persistent storage - data is ephemeral
		storage: type: "emptyDir"

		// Minimal resources for CI/testing
		resources: {
			requests: {
				cpu:    "100m"
				memory: "256Mi"
			}
			limits: {
				cpu:    "500m"
				memory: "1Gi"
			}
		}

		port: 9090

		// Disable config-reload sidecar in ephemeral environments
		configReload: {
			enabled: false
			image: {
				repository: "quay.io/prometheus-operator/prometheus-config-reloader"
				tag:        "v0.89.0"
				digest:     ""
			}
			port: 8080
		}

		serviceAccount: name: "prometheus"
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

		// Ephemeral storage
		storage: type: "emptyDir"

		resources: {
			requests: {
				cpu:    "25m"
				memory: "32Mi"
			}
			limits: {
				cpu:    "100m"
				memory: "128Mi"
			}
		}

		port: 9093

		config: {
			route: {
				receiver:       "default"
				groupBy:        ["alertname", "namespace"]
				groupWait:      "30s"
				groupInterval:  "5m"
				repeatInterval: "4h"
			}
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
				cpu:    "25m"
				memory: "16Mi"
			}
			limits: {
				cpu:    "100m"
				memory: "64Mi"
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
				cpu:    "25m"
				memory: "32Mi"
			}
			limits: {
				cpu:    "100m"
				memory: "128Mi"
			}
		}

		serviceAccount: name: "kube-state-metrics"
	}

	// === Pushgateway ===
	// Disabled in test/CI - batch job metrics are not needed
	pushgateway: {
		enabled: false

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
	security: {
		runAsUser:  65534
		runAsGroup: 65534
	}
}

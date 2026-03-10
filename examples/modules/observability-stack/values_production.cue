// Production values for the Observability Stack.
// Apply with: opm mod build examples/observability-stack -f examples/observability-stack/values_production.cue
//
// Differences from default (values.cue):
// - Prometheus: 2 replicas (HA), 50Gi storage, 30d retention, ingress with TLS
// - Alertmanager: 2 replicas (HA), 10Gi storage
// - Faster scrape intervals (15s)
// - External labels for multi-cluster identification
// - Higher resource allocations across the board
package observability

values: {

	// === Global ===
	global: {
		scrapeInterval:     "15s"
		scrapeTimeout:      "10s"
		evaluationInterval: "15s"

		// External labels help identify this cluster in federated setups
		externalLabels: {
			cluster:     "production"
			environment: "prod"
		}
	}

	// === Prometheus Server ===
	prometheus: {
		image: {
			repository: "quay.io/prometheus/prometheus"
			tag:        "v3.4.1"
			digest:     ""
		}

		// HA: two replicas with stable identity (StatefulSet)
		replicas: 1

		// 30 days retention with size cap
		retention:     "30d"
		retentionSize: "45GB"

		// Larger storage for production workloads
		storage: {
			type: "pvc"
			size: "50Gi"
		}

		// Production resource allocation
		resources: {
			requests: {
				cpu:    "1000m"
				memory: "2Gi"
			}
			limits: {
				cpu:    "4000m"
				memory: "8Gi"
			}
		}

		port: 9090

		// Config-reload sidecar enabled
		configReload: {
			enabled: true
			image: {
				repository: "quay.io/prometheus-operator/prometheus-config-reloader"
				tag:        "v0.89.0"
				digest:     ""
			}
			port: 8080
		}

		// Enable ingress for production access
		ingress: {
			hostname:         "prometheus.example.com"
			path:             "/"
			ingressClassName: "nginx"
			tls: {
				enabled:    true
				secretName: "prometheus-tls"
			}
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

		// HA: two replicas for alert reliability
		replicas: 2

		storage: {
			type: "pvc"
			size: "10Gi"
		}

		resources: {
			requests: {
				cpu:    "100m"
				memory: "128Mi"
			}
			limits: {
				cpu:    "500m"
				memory: "512Mi"
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
				cpu:    "100m"
				memory: "64Mi"
			}
			limits: {
				cpu:    "500m"
				memory: "256Mi"
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
				cpu:    "100m"
				memory: "128Mi"
			}
			limits: {
				cpu:    "500m"
				memory: "512Mi"
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
				cpu:    "50m"
				memory: "64Mi"
			}
			limits: {
				cpu:    "200m"
				memory: "128Mi"
			}
		}
	}

	// === Security ===
	security: {
		runAsUser:  65534
		runAsGroup: 65534
	}
}

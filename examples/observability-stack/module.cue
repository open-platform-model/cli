// Package main defines the Observability Stack module.
// A production-grade monitoring stack based on Prometheus, modeled after the
// prometheus-community/prometheus Helm chart. Demonstrates:
// - Multi-component architecture (5 components, 4 workload types)
// - StatefulSet (prometheus, alertmanager), DaemonSet (node-exporter), Deployment (kube-state-metrics, pushgateway)
// - ConfigMaps with CUE-native config generation via encoding/yaml
// - Conditional components and traits
// - Sidecar containers (configmap-reload)
// - Multiple value overlays (default, production, minimal)
// - Security hardening, health checks, persistent storage
package main

import (
	"opmodel.dev/core@v1"
	schemas "opmodel.dev/schemas@v1"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	modulePath:       "example.com/modules"
	name:             "observability-stack"
	version:          "0.1.0"
	description:      "Production monitoring stack with Prometheus, Alertmanager, Node Exporter, Kube State Metrics, and Pushgateway"
	defaultNamespace: "monitoring"
}

// Schema only - constraints for users, no defaults
#config: {

	// === Global Prometheus Settings ===
	global: {
		// How frequently to scrape targets
		scrapeInterval: string

		// Per-target scrape timeout
		scrapeTimeout: string

		// How frequently to evaluate alerting and recording rules
		evaluationInterval: string

		// External labels applied to all metrics before sending to external systems
		externalLabels?: [string]: string
	}

	// === Prometheus Server ===
	prometheus: {
		// Container image for Prometheus server
		image: schemas.#Image

		// Data retention duration (e.g., "15d", "30d", "90d")
		retention: string

		// Optional: Maximum TSDB size before oldest data is dropped (e.g., "50GB")
		retentionSize?: string

		// Number of replicas (>1 requires StatefulSet for stable identity)
		replicas: int & >=1 & <=10

		// TSDB data storage
		storage: {
			type: "pvc" | "emptyDir"

			// For PVC
			size?:         string
			storageClass?: string
		}

		// Resource requests and limits
		resources?: schemas.#ResourceRequirementsSchema

		// Prometheus server port
		port: int & >0 & <=65535

		// Enable config-reloader sidecar to watch for ConfigMap changes
		configReload: {
			enabled: bool
			image:   schemas.#Image
			// Metrics port for the config-reloader sidecar
			port: int & >0 & <=65535
		}

		// Optional: Ingress configuration for Prometheus UI
		ingress?: {
			hostname:         string
			path:             string | *"/"
			ingressClassName: string | *"nginx"
			tls?: {
				enabled:    bool
				secretName: string
			}
		}

		// Service account name for RBAC (Prometheus needs cluster-wide read access)
		serviceAccount: {
			name: string
		}

		// Extra command-line flags passed to Prometheus (e.g., ["web.enable-lifecycle"])
		extraFlags: [...string]
	}

	// === Alertmanager ===
	alertmanager: {
		// Enable Alertmanager component
		enabled: bool

		// Container image
		image: schemas.#Image

		// Number of replicas
		replicas: int & >=1 & <=5

		// Alert data storage
		storage: {
			type: "pvc" | "emptyDir"

			// For PVC
			size?:         string
			storageClass?: string
		}

		// Resource requests and limits
		resources?: schemas.#ResourceRequirementsSchema

		// Alertmanager web UI and API port
		port: int & >0 & <=65535

		// Alertmanager configuration
		config: {
			// Route tree for alert routing
			route: {
				// Default receiver for unmatched alerts
				receiver: string
				// Group alerts by these labels
				groupBy: [...string]
				// How long to wait before sending a notification for a new group
				groupWait: string
				// How long to wait before sending updated notifications for a group
				groupInterval: string
				// How long to wait before resending a notification
				repeatInterval: string
			}

			// Notification receivers
			receivers: [...{
				name: string
			}]
		}
	}

	// === Node Exporter ===
	nodeExporter: {
		// Enable Node Exporter component
		enabled: bool

		// Container image
		image: schemas.#Image

		// Metrics port
		port: int & >0 & <=65535

		// Resource requests and limits
		resources?: schemas.#ResourceRequirementsSchema

		// Host paths to mount for system metrics collection
		hostPaths: {
			proc: string | *"/proc"
			sys:  string | *"/sys"
		}
	}

	// === Kube State Metrics ===
	kubeStateMetrics: {
		// Enable Kube State Metrics component
		enabled: bool

		// Container image
		image: schemas.#Image

		// Metrics port
		port: int & >0 & <=65535

		// Telemetry port for KSM's own metrics
		telemetryPort: int & >0 & <=65535

		// Resource requests and limits
		resources?: schemas.#ResourceRequirementsSchema

		// Service account name (needs cluster-wide read access to K8s objects)
		serviceAccount: {
			name: string
		}
	}

	// === Pushgateway ===
	pushgateway: {
		// Enable Pushgateway component
		enabled: bool

		// Container image
		image: schemas.#Image

		// Web UI and API port
		port: int & >0 & <=65535

		// Resource requests and limits
		resources?: schemas.#ResourceRequirementsSchema
	}

	// === Security ===
	// Shared security settings applied to all components
	security: {
		// Run as non-root user (65534 = nobody, standard for Prometheus ecosystem)
		runAsUser:  int
		runAsGroup: int
	}
}

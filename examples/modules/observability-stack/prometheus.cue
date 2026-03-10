// Prometheus server component: the core metrics collection and storage engine.
// StatefulSet with TSDB persistent storage, configmap-reload sidecar,
// prometheus.yml ConfigMap generated from pure CUE, and optional Ingress.
package observability

import (
	"encoding/yaml"
	resources_workload "opmodel.dev/resources/workload@v1"
	resources_storage "opmodel.dev/resources/storage@v1"
	resources_config "opmodel.dev/resources/config@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
	traits_security "opmodel.dev/traits/security@v1"
)

// _prometheusConfig builds the prometheus.yml configuration as a typed CUE struct.
// It is serialized to YAML via encoding/yaml.Marshal() for the ConfigMap.
let _prometheusConfig = {
	global: {
		scrape_interval:     #config.global.scrapeInterval
		scrape_timeout:      #config.global.scrapeTimeout
		evaluation_interval: #config.global.evaluationInterval
		if #config.global.externalLabels != _|_ {
			external_labels: #config.global.externalLabels
		}
	}

	// Scrape configurations for each component
	scrape_configs: [
		// Self-scrape: Prometheus's own metrics
		{
			job_name: "prometheus"
			static_configs: [{
				targets: ["localhost:\(#config.prometheus.port)"]
			}]
			metrics_path: "/metrics"
		},

		// Node Exporter: host-level metrics (CPU, memory, disk, network)
		if #config.nodeExporter.enabled {
			{
				job_name: "node-exporter"
				static_configs: [{
					targets: ["node-exporter:\(#config.nodeExporter.port)"]
					labels: component: "node-exporter"
				}]
				metrics_path: "/metrics"
			}
		},

		// Kube State Metrics: Kubernetes object state metrics
		if #config.kubeStateMetrics.enabled {
			{
				job_name: "kube-state-metrics"
				static_configs: [{
					targets: ["kube-state-metrics:\(#config.kubeStateMetrics.port)"]
					labels: component: "kube-state-metrics"
				}]
				metrics_path: "/metrics"
			}
		},

		// Pushgateway: batch job metrics
		if #config.pushgateway.enabled {
			{
				job_name:     "pushgateway"
				honor_labels: true
				static_configs: [{
					targets: ["pushgateway:\(#config.pushgateway.port)"]
					labels: component: "pushgateway"
				}]
				metrics_path: "/metrics"
			}
		},
	]

	// Rule files loaded from the config volume
	rule_files: [
		"/etc/prometheus/alerting_rules.yml",
		"/etc/prometheus/recording_rules.yml",
	]

	// Alertmanager discovery
	if #config.alertmanager.enabled {
		alerting: alertmanagers: [{
			static_configs: [{
				targets: ["alertmanager:\(#config.alertmanager.port)"]
			}]
		}]
	}
}

// Alerting rules as a CUE struct (placeholder groups for user customization)
let _alertingRules = {
	groups: [{
		name: "node-exporter"
		rules: [{
			alert: "HighNodeCPU"
			expr:  "100 - (avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 80"
			"for": "5m"
			labels: severity: "warning"
			annotations: {
				summary:     "High CPU usage on {{ $labels.instance }}"
				description: "CPU usage is above 80% for more than 5 minutes."
			}
		}, {
			alert: "HighNodeMemory"
			expr:  "(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100 > 85"
			"for": "5m"
			labels: severity: "warning"
			annotations: {
				summary:     "High memory usage on {{ $labels.instance }}"
				description: "Memory usage is above 85% for more than 5 minutes."
			}
		}, {
			alert: "NodeDiskSpaceLow"
			expr:  "(1 - node_filesystem_avail_bytes{fstype!=\"tmpfs\"} / node_filesystem_size_bytes{fstype!=\"tmpfs\"}) * 100 > 90"
			"for": "10m"
			labels: severity: "critical"
			annotations: {
				summary:     "Disk space low on {{ $labels.instance }}"
				description: "Disk usage is above 90% on {{ $labels.mountpoint }}."
			}
		}]
	}, {
		name: "prometheus"
		rules: [{
			alert: "PrometheusTargetDown"
			expr:  "up == 0"
			"for": "3m"
			labels: severity: "critical"
			annotations: {
				summary:     "Target {{ $labels.job }}/{{ $labels.instance }} is down"
				description: "Prometheus target has been unreachable for more than 3 minutes."
			}
		}]
	}]
}

// Recording rules for pre-computed aggregations
let _recordingRules = {
	groups: [{
		name: "aggregations"
		rules: [{
			record: "job:up:sum"
			expr:   "sum by(job) (up)"
		}, {
			record: "instance:node_cpu_utilisation:rate5m"
			expr:   "1 - avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m]))"
		}, {
			record: "instance:node_memory_utilisation:ratio"
			expr:   "1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes"
		}]
	}]
}

#components: {

	/////////////////////////////////////////////////////////////////
	//// Prometheus Server - Stateful Metrics Collection Engine
	/////////////////////////////////////////////////////////////////

	prometheus: {
		resources_workload.#Container
		resources_storage.#Volumes
		resources_config.#ConfigMaps
		if #config.prometheus.configReload.enabled {
			traits_workload.#SidecarContainers
		}
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_network.#Expose
		if #config.prometheus.ingress != _|_ {
			traits_network.#HttpRoute
		}
		traits_security.#SecurityContext
		traits_security.#WorkloadIdentity

		metadata: labels: "core.opmodel.dev/workload-type": "stateful"

		spec: {
			scaling: count: #config.prometheus.replicas

			restartPolicy: "Always"

			// === ConfigMaps ===
			// Prometheus configuration generated from pure CUE structs
			configMaps: {
				"prometheus-config": {
					data: {
						"prometheus.yml":      yaml.Marshal(_prometheusConfig)
						"alerting_rules.yml":  yaml.Marshal(_alertingRules)
						"recording_rules.yml": yaml.Marshal(_recordingRules)
					}
				}
			}

			// === Main Container: Prometheus Server ===
			container: {
				name:  "prometheus"
				image: #config.prometheus.image

				ports: {
					http: {
						name:       "http"
						targetPort: #config.prometheus.port
						protocol:   "TCP"
					}
				}

				// Prometheus command-line arguments
				args: [
					"--config.file=/etc/prometheus/prometheus.yml",
					"--storage.tsdb.path=/data",
					"--storage.tsdb.retention.time=\(#config.prometheus.retention)",
					if #config.prometheus.retentionSize != _|_ {
						"--storage.tsdb.retention.size=\(#config.prometheus.retentionSize)"
					},
					for flag in #config.prometheus.extraFlags {
						"--\(flag)"
					},
				]

				env: {
					// Expose pod name for HA setups (external_labels)
					POD_NAME: {
						name:  "POD_NAME"
						value: "prometheus"
					}
				}

				volumeMounts: {
					"prometheus-data": volumes["prometheus-data"] & {
						mountPath: "/data"
					}
					"prometheus-config": volumes["prometheus-config"] & {
						mountPath: "/etc/prometheus"
						readOnly:  true
					}
				}

				if #config.prometheus.resources != _|_ {
					resources: #config.prometheus.resources
				}

				// === Health Checks ===
				// Startup probe: give Prometheus time to load TSDB WAL replay
				// Allows up to 5 minutes (failureThreshold=30 x periodSeconds=10)
				startupProbe: {
					httpGet: {
						path: "/-/ready"
						port: #config.prometheus.port
					}
					periodSeconds:    10
					timeoutSeconds:   5
					failureThreshold: 30
				}
				// Liveness probe: restart if Prometheus becomes unhealthy
				livenessProbe: {
					httpGet: {
						path: "/-/healthy"
						port: #config.prometheus.port
					}
					periodSeconds:    15
					timeoutSeconds:   10
					failureThreshold: 3
				}
				// Readiness probe: only route traffic when ready to serve queries
				readinessProbe: {
					httpGet: {
						path: "/-/ready"
						port: #config.prometheus.port
					}
					periodSeconds:    5
					timeoutSeconds:   4
					failureThreshold: 3
				}
			}

			// === Sidecar: Config Reloader ===
			// Watches /etc/prometheus for ConfigMap changes and triggers a reload
			// via POST to /-/reload (requires --web.enable-lifecycle flag)
			if #config.prometheus.configReload.enabled {
				sidecarContainers: [{
					name:  "config-reloader"
					image: #config.prometheus.configReload.image

					args: [
						"--watched-dir=/etc/prometheus",
						"--reload-url=http://127.0.0.1:\(#config.prometheus.port)/-/reload",
					]

					ports: {
						reloader: {
							targetPort: #config.prometheus.configReload.port
							protocol:   "TCP"
						}
					}

					volumeMounts: {
						"prometheus-config": volumes["prometheus-config"] & {
							mountPath: "/etc/prometheus"
							readOnly:  true
						}
					}

					resources: {
						requests: {
							cpu:    "50m"
							memory: "32Mi"
						}
						limits: {
							cpu:    "100m"
							memory: "64Mi"
						}
					}
				}]
			}

			// === Service Exposure ===
			expose: {
				ports: http: container.ports.http & {
					exposedPort: #config.prometheus.port
				}
				type: "ClusterIP"
			}

			// === Ingress (conditional) ===
			if #config.prometheus.ingress != _|_ {
				httpRoute: {
					hostnames: [#config.prometheus.ingress.hostname]
					rules: [{
						matches: [{
							path: {
								type:  "Prefix"
								value: #config.prometheus.ingress.path
							}
						}]
						backendPort: #config.prometheus.port
					}]
					if #config.prometheus.ingress.ingressClassName != _|_ {
						ingressClassName: #config.prometheus.ingress.ingressClassName
					}
					if #config.prometheus.ingress.tls != _|_ && #config.prometheus.ingress.tls.enabled {
						tls: {
							mode: "Terminate"
							certificateRef: name: #config.prometheus.ingress.tls.secretName
						}
					}
				}
			}

			// === Security ===
			securityContext: {
				runAsNonRoot:             true
				runAsUser:                #config.security.runAsUser
				runAsGroup:               #config.security.runAsGroup
				readOnlyRootFilesystem:   false
				allowPrivilegeEscalation: false
				capabilities: drop: ["ALL"]
			}

			// === Workload Identity ===
			workloadIdentity: {
				name:           #config.prometheus.serviceAccount.name
				automountToken: true // Prometheus needs API access for service discovery
			}

			// === Volumes ===
			volumes: {
				// TSDB data volume
				"prometheus-data": {
					name: "prometheus-data"

					if #config.prometheus.storage.type == "pvc" {
						persistentClaim: {
							size: #config.prometheus.storage.size
							if #config.prometheus.storage.storageClass != _|_ {
								storageClass: #config.prometheus.storage.storageClass
							}
						}
					}

					if #config.prometheus.storage.type == "emptyDir" {
						emptyDir: {}
					}
				}

				// ConfigMap volume: mounts prometheus.yml, alerting/recording rules
				"prometheus-config": {
					name: "prometheus-config"
					configMap: configMaps["prometheus-config"]
				}
			}
		}
	}
}

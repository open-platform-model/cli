// Components defines the mc-monitor workload.
// Single stateless component that polls Minecraft server status and
// exports metrics via Prometheus HTTP scrape or OpenTelemetry gRPC push.
package mc_monitor

import (
	"strings"

	resources_workload "opmodel.dev/resources/workload@v1"
	traits_workload    "opmodel.dev/traits/workload@v1"
	traits_network     "opmodel.dev/traits/network@v1"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	/////////////////////////////////////////////////////////////////
	//// Monitor — Stateless Metrics Exporter
	/////////////////////////////////////////////////////////////////

	monitor: {
		resources_workload.#Container
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_workload.#UpdateStrategy

		// Only embed Expose trait in Prometheus mode — OTel is push-based (no inbound port).
		if #config.prometheus != _|_ {
			traits_network.#Expose
		}

		metadata: labels: "core.opmodel.dev/workload-type": "stateless"

		spec: {
			// Single replica — one exporter monitors the whole fleet
			scaling: count: 1

			restartPolicy: "Always"

			// Recreate strategy — stateless, no need for rolling updates
			updateStrategy: type: "Recreate"

			// === Main Container: mc-monitor ===
			container: {
				name:  "mc-monitor"
				image: #config.image

				// Select subcommand based on export mode
				if #config.prometheus != _|_ {
					command: ["/mc-monitor", "export-for-prometheus"]
				}
				if #config.otel != _|_ {
					command: ["/mc-monitor", "collect-otel"]
				}

				// Prometheus mode: expose the metrics HTTP port
				if #config.prometheus != _|_ {
					ports: {
						metrics: {
							targetPort: #config.prometheus.port
							protocol:   "TCP"
						}
					}
				}

				env: {
					// Java servers — space-separated host:port list
					EXPORT_SERVERS: {
						name: "EXPORT_SERVERS"
						value: strings.Join([ for s in #config.javaServers {
							"\(s.host):\(s.port)"
						}], ",")
					}

					// Bedrock servers — space-separated host:port list (conditional)
					if #config.bedrockServers != _|_ {
						EXPORT_BEDROCK_SERVERS: {
							name: "EXPORT_BEDROCK_SERVERS"
					value: strings.Join([ for s in #config.bedrockServers {
							"\(s.host):\(s.port)"
						}], ",")
						}
					}

					// Per-server check timeout
					TIMEOUT: {
						name:  "TIMEOUT"
						value: #config.timeout
					}

					// Prometheus-specific env vars
					if #config.prometheus != _|_ {
						EXPORT_PORT: {
							name:  "EXPORT_PORT"
							value: "\(#config.prometheus.port)"
						}
					}

					// OTel-specific env vars
					if #config.otel != _|_ {
						EXPORT_OTEL_COLLECTOR_ENDPOINT: {
							name:  "EXPORT_OTEL_COLLECTOR_ENDPOINT"
							value: #config.otel.collectorEndpoint
						}
						EXPORT_OTEL_COLLECTOR_TIMEOUT: {
							name:  "EXPORT_OTEL_COLLECTOR_TIMEOUT"
							value: #config.otel.collectorTimeout
						}
						EXPORT_INTERVAL: {
							name:  "EXPORT_INTERVAL"
							value: #config.otel.interval
						}
					}
				}

				if #config.resources != _|_ {
					resources: #config.resources
				}
			}

			// Network exposure (Prometheus mode only)
			if #config.prometheus != _|_ {
				expose: {
					ports: {
						metrics: {
							targetPort:  #config.prometheus.port
							protocol:    "TCP"
							exposedPort: #config.prometheus.port
						}
					}
					type: #config.serviceType
				}
			}
		}
	}
}

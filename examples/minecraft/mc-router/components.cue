// Components defines the mc-router workload.
// Single stateless component that routes Minecraft TCP connections by hostname.
package main

import (
	resources_workload "opmodel.dev/resources/workload@v1"
	resources_security "opmodel.dev/resources/security@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
	traits_security "opmodel.dev/traits/security@v1"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	/////////////////////////////////////////////////////////////////
	//// Router - Stateless TCP Hostname Router
	/////////////////////////////////////////////////////////////////

	router: {
		resources_workload.#Container
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_workload.#UpdateStrategy
		traits_network.#Expose
		traits_security.#WorkloadIdentity

		metadata: labels: "core.opmodel.dev/workload-type": "stateless"

		spec: {
			// Single replica by default; use scaling config for more
			scaling: count: 1

			restartPolicy: "Always"

			// Recreate strategy - simple stateless router
			updateStrategy: type: "Recreate"

			// Workload identity (service account) - needed for auto-scale RBAC
			workloadIdentity: {
				name:           "mc-router"
				automountToken: true
			}

			// === Main Container: mc-router ===
			container: {
				name:  "mc-router"
				image: #config.router.image

				ports: {
					minecraft: {
						targetPort: #config.port
						protocol:   "TCP"
					}
					if #config.router.api != _|_ && #config.router.api.enabled && #config.router.api.port != _|_ {
						api: {
							targetPort: #config.router.api.port
							protocol:   "TCP"
						}
					}
				}

				env: {
					// Port configuration
					PORT: {
						name:  "PORT"
						value: "\(#config.port)"
					}

					// Connection rate limit
					if #config.router.connectionRateLimit != _|_ {
						CONNECTION_RATE_LIMIT: {
							name:  "CONNECTION_RATE_LIMIT"
							value: "\(#config.router.connectionRateLimit)"
						}
					}

					// Debug mode
					if #config.router.debug != _|_ {
						DEBUG: {
							name:  "DEBUG"
							value: "\(#config.router.debug)"
						}
					}

					// Simplify SRV
					if #config.router.simplifySrv != _|_ {
						SIMPLIFY_SRV: {
							name:  "SIMPLIFY_SRV"
							value: "\(#config.router.simplifySrv)"
						}
					}

					// PROXY protocol
					if #config.router.useProxyProtocol != _|_ {
						USE_PROXY_PROTOCOL: {
							name:  "USE_PROXY_PROTOCOL"
							value: "\(#config.router.useProxyProtocol)"
						}
					}

					// Default server (host:port format)
					if #config.router.defaultServer != _|_ {
						DEFAULT: {
							name:  "DEFAULT"
							value: "\(#config.router.defaultServer.host):\(#config.router.defaultServer.port)"
						}
					}

					// Auto-scale up
					if #config.router.autoScale != _|_ && #config.router.autoScale.up != _|_ {
						AUTO_SCALE_UP: {
							name:  "AUTO_SCALE_UP"
							value: "\(#config.router.autoScale.up.enabled)"
						}
					}

					// Auto-scale down
					if #config.router.autoScale != _|_ && #config.router.autoScale.down != _|_ {
						AUTO_SCALE_DOWN: {
							name:  "AUTO_SCALE_DOWN"
							value: "\(#config.router.autoScale.down.enabled)"
						}
						if #config.router.autoScale.down.after != _|_ {
							AUTO_SCALE_DOWN_AFTER: {
								name:  "AUTO_SCALE_DOWN_AFTER"
								value: #config.router.autoScale.down.after
							}
						}
					}

					// Metrics backend
					if #config.router.metrics != _|_ {
						METRICS_BACKEND: {
							name:  "METRICS_BACKEND"
							value: #config.router.metrics.backend
						}
					}

					// API port
					if #config.router.api != _|_ && #config.router.api.enabled && #config.router.api.port != _|_ {
						API_BINDING: {
							name:  "API_BINDING"
							value: ":\(#config.router.api.port)"
						}
					}
				}

				// Build --mapping flags from static mappings
				if #config.router.mappings != _|_ {
					args: [ for m in #config.router.mappings {
						if m.port != _|_ {
							"--mapping=\(m.externalHostname)=\(m.host):\(m.port)"
						}
						if m.port == _|_ {
							"--mapping=\(m.externalHostname)=\(m.host)"
						}
					}]
				}

				if #config.resources != _|_ {
					resources: #config.resources
				}
			}

		// === Network Exposure ===
		expose: {
			ports: {
				minecraft: {
					targetPort:  #config.port
					protocol:    "TCP"
					exposedPort: #config.port
				}
				if #config.router.api != _|_ && #config.router.api.enabled && #config.router.api.port != _|_ {
					api: {
						targetPort:  #config.router.api.port
						protocol:    "TCP"
						exposedPort: #config.router.api.port
					}
				}
			}
			type: #config.serviceType
		}
	}
}

	/////////////////////////////////////////////////////////////////
	//// RBAC - ClusterRole + ClusterRoleBinding
	//// Grants mc-router permission to:
	////   - Always:      watch/list Services (service discovery for routing)
	////   - Conditional: watch/list/get/update/patch StatefulSets (auto-scale)
	/////////////////////////////////////////////////////////////////

	rbac: {
		resources_security.#Role

		spec: role: {
			name:  "mc-router"
			scope: "cluster"

			rules: [
				// Core: service discovery — mc-router watches Services labelled with
				// routing annotations to build its hostname→backend map.
				{
					apiGroups: [""]
					resources: ["services"]
					verbs: ["watch", "list"]
				},
				// StatefulSet scaling — needed when autoScale.up/down is enabled to wake/sleep
				// backend servers. Included unconditionally; mc-router only invokes scale APIs
				// when AUTO_SCALE_UP/AUTO_SCALE_DOWN env vars are set.
				{
					apiGroups: ["apps"]
					resources: ["statefulsets", "statefulsets/scale"]
					verbs: ["watch", "list", "get", "update", "patch"]
				},
			]

			// Bind to the mc-router workload identity (ServiceAccount).
			subjects: [{
				name: "mc-router"
			}]
		}
	}
}

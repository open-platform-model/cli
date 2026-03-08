// Components defines the Velocity proxy workload.
// Single stateless container — no persistent storage needed.
package velocity

import (
	resources_workload "opmodel.dev/resources/workload@v1"
	resources_storage "opmodel.dev/resources/storage@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
	traits_security "opmodel.dev/traits/security@v1"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	/////////////////////////////////////////////////////////////////
	//// Velocity — Stateless Minecraft Proxy
	/////////////////////////////////////////////////////////////////

	proxy: {
		resources_workload.#Container
		resources_storage.#Volumes
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_workload.#UpdateStrategy
		traits_workload.#GracefulShutdown
		traits_network.#Expose
		traits_security.#SecurityContext

		metadata: labels: "core.opmodel.dev/workload-type": "stateless"

		spec: {
			// Stateless proxy — can run multiple replicas
			scaling: count: 1

			restartPolicy: "Always"

			updateStrategy: type: "RollingUpdate"

			// Allow in-flight connections to drain before termination
			gracefulShutdown: terminationGracePeriodSeconds: 30

			// Non-root security context.
			// fsGroup: 3000 sets group ownership on mounted volumes so the
			// process (GID 3000) can write to the /server emptyDir at startup.
			securityContext: {
				runAsNonRoot:             true
				runAsUser:                1000
				runAsGroup:               3000
				fsGroup:                  3000
				readOnlyRootFilesystem:   true
				allowPrivilegeEscalation: false
				capabilities: drop: ["ALL"]
			}

			// === Main Container: Velocity Proxy ===
			container: {
				name:  "proxy"
				image: #config.image

				ports: {
					minecraft: {
						targetPort: #config.bindPort
						protocol:   "TCP"
					}
				}

				env: {
					TYPE: {
						name:  "TYPE"
						value: #config.type
					}
					MOTD: {
						name:  "MOTD"
						value: #config.motd
					}
					ONLINE_MODE: {
						name:  "ONLINE_MODE"
						value: "\(#config.onlineMode)"
					}
					MAX_PLAYERS: {
						name:  "MAX_PLAYERS"
						value: "\(#config.maxPlayers)"
					}
					VELOCITY_FORWARDING_MODE: {
						name:  "VELOCITY_FORWARDING_MODE"
						value: #config.forwardingMode
					}
				if #config.forwardingSecret != _|_ {
					VELOCITY_FORWARDING_SECRET: {
						name:  "VELOCITY_FORWARDING_SECRET"
						value: #config.forwardingSecret
					}
				}
			}

			// Mount the writable emptyDir so the Velocity JAR can be downloaded
			// into /server at startup (image working directory, owned by root by default).
			volumeMounts: {
				"server-data": volumes["server-data"] & {
					mountPath: "/server"
				}
			}
		}

		// === Volumes ===
		volumes: {
			"server-data": {
				name:     "server-data"
				emptyDir: {}
			}
		}

		// === Network Exposure ===
		expose: {
				ports: minecraft: {
					targetPort:  #config.bindPort
					protocol:    "TCP"
					exposedPort: #config.bindPort
				}
				type: "LoadBalancer"
			}
		}
	}
}

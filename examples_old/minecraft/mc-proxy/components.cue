// Components defines the Minecraft proxy workload.
// Single stateful component with persistent data for proxy config and plugins.
package main

import (
	"strings"

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
	//// Proxy - Stateful BungeeCord/Waterfall/Velocity Server
	/////////////////////////////////////////////////////////////////

	proxy: {
		resources_workload.#Container
		resources_storage.#Volumes
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_workload.#UpdateStrategy
		traits_network.#Expose
		traits_security.#SecurityContext

		metadata: labels: "core.opmodel.dev/workload-type": "stateful"

		spec: {
			// Single replica by default - can be scaled for network proxying
			scaling: count: 1

			restartPolicy: "Always"

			// Recreate strategy - avoid split-brain with config dir
			updateStrategy: type: "Recreate"

			// === Security Context ===
			if #config.securityContext != _|_ {
				securityContext: #config.securityContext
			}
			if #config.securityContext == _|_ {
				securityContext: {
					runAsNonRoot:             true
					runAsUser:                1000
					runAsGroup:               3000
					readOnlyRootFilesystem:   true
					allowPrivilegeEscalation: false
					capabilities: drop: ["ALL"]
				}
			}

			// === Main Container: Proxy Server ===
			container: {
				name:  "proxy"
				image: #config.proxy.image

				ports: {
					proxy: {
						targetPort: #config.port
						protocol:   "TCP"
					}
				if #config.proxy.rcon != _|_ {
					if #config.proxy.rcon.enabled {
						rcon: {
							name:       "rcon"
							targetPort: #config.proxy.rcon.port
							protocol:   "TCP"
						}
					}
				}
					if #config.proxy.extraPorts != _|_ {
						for ep in #config.proxy.extraPorts {
							(ep.name): {
								name:       ep.name
								targetPort: ep.containerPort
								protocol:   ep.protocol
							}
						}
					}
				}

				env: {
					// Proxy type
					TYPE: {
						name:  "TYPE"
						value: #config.proxy.type
					}

					// === JVM Configuration ===
					if #config.proxy.memory != _|_ {
						MEMORY: {
							name:  "MEMORY"
							value: #config.proxy.memory
						}
					}
					if #config.proxy.jvmOpts != _|_ {
						JVM_OPTS: {
							name:  "JVM_OPTS"
							value: #config.proxy.jvmOpts
						}
					}
					if #config.proxy.jvmXXOpts != _|_ {
						JVM_XX_OPTS: {
							name:  "JVM_XX_OPTS"
							value: #config.proxy.jvmXXOpts
						}
					}

					// Online mode
					if #config.proxy.onlineMode != _|_ {
						ONLINE_MODE: {
							name:  "ONLINE_MODE"
							value: "\(#config.proxy.onlineMode)"
						}
					}

					// Plugins (newline-separated URLs)
					if #config.proxy.plugins != _|_ {
						PLUGINS: {
							name:  "PLUGINS"
							value: strings.Join(#config.proxy.plugins, "\n")
						}
					}

					// Config file path
					if #config.proxy.configFilePath != _|_ {
						CFG_FILE: {
							name:  "CFG_FILE"
							value: #config.proxy.configFilePath
						}
					}

					// Inline config content
					if #config.proxy.configContent != _|_ {
						CFG_CONTENT: {
							name:  "CFG_CONTENT"
							value: #config.proxy.configContent
						}
					}

					// === Type-Specific Configuration ===
					// WATERFALL
					if #config.proxy.waterfallVersion != _|_ {
						WATERFALL_VERSION: {
							name:  "WATERFALL_VERSION"
							value: #config.proxy.waterfallVersion
						}
					}
					if #config.proxy.waterfallBuildId != _|_ {
						WATERFALL_BUILD_ID: {
							name:  "WATERFALL_BUILD_ID"
							value: #config.proxy.waterfallBuildId
						}
					}

					// VELOCITY
					if #config.proxy.velocityVersion != _|_ {
						VELOCITY_VERSION: {
							name:  "VELOCITY_VERSION"
							value: #config.proxy.velocityVersion
						}
					}

					// CUSTOM
					if #config.proxy.jarUrl != _|_ {
						BUNGEE_JAR_URL: {
							name:  "BUNGEE_JAR_URL"
							value: #config.proxy.jarUrl
						}
					}
					if #config.proxy.jarFile != _|_ {
						BUNGEE_JAR_FILE: {
							name:  "BUNGEE_JAR_FILE"
							value: #config.proxy.jarFile
						}
					}

				// === RCON Configuration ===
				if #config.proxy.rcon != _|_ {
					if #config.proxy.rcon.enabled {
						ENABLE_RCON: {
							name:  "ENABLE_RCON"
							value: "true"
						}
						RCON_PASSWORD: {
							name: "RCON_PASSWORD"
							from: #config.proxy.rcon.password
						}
						RCON_PORT: {
							name:  "RCON_PORT"
							value: "\(#config.proxy.rcon.port)"
						}
					}
				}
				}

				volumeMounts: {
					data: volumes.data & {
						mountPath: "/server"
					}
				}

				if #config.resources != _|_ {
					resources: #config.resources
				}
			}

			// === Network Exposure ===
			expose: {
				ports: proxy: {
					targetPort:  #config.port
					protocol:    "TCP"
					exposedPort: #config.port
				}
				type: #config.serviceType
			}

			// === Volumes ===
			volumes: {
				data: {
					name: "data"
					if #config.storage.data.type == "pvc" {
						persistentClaim: {
							size: #config.storage.data.size
							if #config.storage.data.storageClass != _|_ {
								storageClass: #config.storage.data.storageClass
							}
						}
					}
					if #config.storage.data.type == "hostPath" {
						hostPath: {
							path: #config.storage.data.path
							type: #config.storage.data.hostPathType
						}
					}
					if #config.storage.data.type == "emptyDir" {
						emptyDir: {}
					}
				}
			}
		}
	}
}

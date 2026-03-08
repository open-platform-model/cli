// Components defines the Minecraft Bedrock Edition server workload.
// Single stateful component with persistent data and UDP networking.
// Bedrock Edition does not support RCON or backup sidecars.
package main

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
	//// Minecraft Bedrock - Stateful Game Server
	/////////////////////////////////////////////////////////////////

	server: {
		resources_workload.#Container
		resources_storage.#Volumes
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_workload.#UpdateStrategy
		traits_network.#Expose
		traits_security.#SecurityContext

		metadata: labels: "core.opmodel.dev/workload-type": "stateful"

		spec: {
			// Single replica - Minecraft servers don't support horizontal scaling
			scaling: count: 1

			restartPolicy: "Always"

			// Recreate strategy - Minecraft cannot do rolling updates
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

			// === Main Container: Bedrock Server ===
			container: {
				name:  "bedrock"
				image: #config.server.image

				ports: {
					bedrock: {
						targetPort: #config.server.serverPort
						protocol:   "UDP"
					}
				}

				env: {
					// Required: EULA acceptance
					EULA: {
						name:  "EULA"
						value: "\(#config.server.eula)"
					}

					// Server version
					VERSION: {
						name:  "VERSION"
						value: #config.server.version
					}

					// Game settings
					DIFFICULTY: {
						name:  "DIFFICULTY"
						value: #config.server.difficulty
					}
					GAMEMODE: {
						name:  "GAMEMODE"
						value: #config.server.gameMode
					}
					MAX_PLAYERS: {
						name:  "MAX_PLAYERS"
						value: "\(#config.server.maxPlayers)"
					}
					DEFAULT_PLAYER_PERMISSION_LEVEL: {
						name:  "DEFAULT_PLAYER_PERMISSION_LEVEL"
						value: #config.server.defaultPermission
					}
					SERVER_PORT: {
						name:  "SERVER_PORT"
						value: "\(#config.server.serverPort)"
					}

					// Optional: Server name
					if #config.server.serverName != _|_ {
						SERVER_NAME: {
							name:  "SERVER_NAME"
							value: #config.server.serverName
						}
					}

					// Optional: Online mode (Xbox Live authentication)
					if #config.server.onlineMode != _|_ {
						ONLINE_MODE: {
							name:  "ONLINE_MODE"
							value: "\(#config.server.onlineMode)"
						}
					}

					// Optional: View distance
					if #config.server.viewDistance != _|_ {
						VIEW_DISTANCE: {
							name:  "VIEW_DISTANCE"
							value: "\(#config.server.viewDistance)"
						}
					}

					// Optional: Tick distance
					if #config.server.tickDistance != _|_ {
						TICK_DISTANCE: {
							name:  "TICK_DISTANCE"
							value: "\(#config.server.tickDistance)"
						}
					}

					// Optional: Player idle timeout
					if #config.server.playerIdleTimeout != _|_ {
						PLAYER_IDLE_TIMEOUT: {
							name:  "PLAYER_IDLE_TIMEOUT"
							value: "\(#config.server.playerIdleTimeout)"
						}
					}

					// Optional: Level type
					if #config.server.levelType != _|_ {
						LEVEL_TYPE: {
							name:  "LEVEL_TYPE"
							value: #config.server.levelType
						}
					}

					// Optional: Level name
					if #config.server.levelName != _|_ {
						LEVEL_NAME: {
							name:  "LEVEL_NAME"
							value: #config.server.levelName
						}
					}

					// Optional: Level seed
					if #config.server.levelSeed != _|_ {
						LEVEL_SEED: {
							name:  "LEVEL_SEED"
							value: #config.server.levelSeed
						}
					}

					// Optional: Texture pack required
					if #config.server.texturepackRequired != _|_ {
						TEXTUREPACK_REQUIRED: {
							name:  "TEXTUREPACK_REQUIRED"
							value: "\(#config.server.texturepackRequired)"
						}
					}

					// Optional: Max threads
					if #config.server.maxThreads != _|_ {
						MAX_THREADS: {
							name:  "MAX_THREADS"
							value: "\(#config.server.maxThreads)"
						}
					}

					// Optional: Cheats
					if #config.server.cheats != _|_ {
						CHEATS: {
							name:  "CHEATS"
							value: "\(#config.server.cheats)"
						}
					}

					// Optional: Emit server telemetry
					if #config.server.emitServerTelemetry != _|_ {
						EMIT_SERVER_TELEMETRY: {
							name:  "EMIT_SERVER_TELEMETRY"
							value: "\(#config.server.emitServerTelemetry)"
						}
					}

					// Optional: Enable LAN visibility
					if #config.server.enableLanVisibility != _|_ {
						ENABLE_LAN_VISIBILITY: {
							name:  "ENABLE_LAN_VISIBILITY"
							value: "\(#config.server.enableLanVisibility)"
						}
					}

					// Optional: Whitelist
					if #config.server.whitelist != _|_ {
						WHITE_LIST: {
							name:  "WHITE_LIST"
							value: "\(#config.server.whitelist)"
						}
					}

					// Optional: Whitelisted users (comma-separated)
					if #config.server.whitelistUsers != _|_ {
						WHITE_LIST_USERS: {
							name:  "WHITE_LIST_USERS"
							value: #config.server.whitelistUsers
						}
					}

					// Optional: Operator XUIDs (comma-separated)
					if #config.server.ops != _|_ {
						OPS: {
							name:  "OPS"
							value: #config.server.ops
						}
					}

					// Optional: Member XUIDs (comma-separated)
					if #config.server.members != _|_ {
						MEMBERS: {
							name:  "MEMBERS"
							value: #config.server.members
						}
					}

					// Optional: Visitor XUIDs (comma-separated)
					if #config.server.visitors != _|_ {
						VISITORS: {
							name:  "VISITORS"
							value: #config.server.visitors
						}
					}
				}

				volumeMounts: {
					data: volumes.data & {
						mountPath: "/data"
					}
				}

				if #config.resources != _|_ {
					resources: #config.resources
				}
			}

			// === Network Exposure ===
			expose: {
				ports: bedrock: container.ports.bedrock & {
					exposedPort: #config.server.serverPort
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

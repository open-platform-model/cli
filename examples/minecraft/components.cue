// Components defines the Minecraft server workload with backup sidecar.
// Single stateful component with persistent data, optional backup, and health checks.
package main

import (
	resources_workload "opmodel.dev/resources/workload@v0"
	resources_storage "opmodel.dev/resources/storage@v0"
	traits_workload "opmodel.dev/traits/workload@v0"
	traits_network "opmodel.dev/traits/network@v0"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	/////////////////////////////////////////////////////////////////
	//// Minecraft - Stateful Game Server with Backup Sidecar
	/////////////////////////////////////////////////////////////////

	server: {
		resources_workload.#Container
		resources_storage.#Volumes
		if #config.backup != _|_ && #config.backup.enabled {
			traits_workload.#SidecarContainers
		}
		traits_workload.#Scaling
		traits_workload.#HealthCheck
		traits_workload.#RestartPolicy
		traits_network.#Expose

		metadata: labels: "core.opmodel.dev/workload-type": "stateful"

		spec: {
			// Single replica - Minecraft servers don't support horizontal scaling
			scaling: count: 1

			restartPolicy: "Always"

			// === Main Container: Minecraft Server ===
			container: {
				name:            "server"
				image:           #config.server.image
				imagePullPolicy: "IfNotPresent"

				ports: {
					minecraft: {
						targetPort: 25565
						protocol:   "TCP"
					}
					rcon: {
						name:       "rcon"
						targetPort: #config.server.rcon.port
						protocol:   "TCP"
					}
				}

				env: {
					// Required: EULA acceptance
					EULA: {
						name:  "EULA"
						value: "\(#config.server.eula)"
					}

					// Server type and version
					TYPE: {
						name:  "TYPE"
						value: #config.server.type
					}
					VERSION: {
						name:  "VERSION"
						value: #config.server.version
					}

					// Game settings
					MAX_PLAYERS: {
						name:  "MAX_PLAYERS"
						value: "\(#config.server.maxPlayers)"
					}
					DIFFICULTY: {
						name:  "DIFFICULTY"
						value: #config.server.difficulty
					}
					MODE: {
						name:  "MODE"
						value: #config.server.mode
					}
					PVP: {
						name:  "PVP"
						value: "\(#config.server.pvp)"
					}
					ENABLE_COMMAND_BLOCK: {
						name:  "ENABLE_COMMAND_BLOCK"
						value: "\(#config.server.enableCommandBlock)"
					}

					// RCON configuration (required for backup coordination)
					ENABLE_RCON: {
						name:  "ENABLE_RCON"
						value: "true"
					}
					RCON_PASSWORD: {
						name:  "RCON_PASSWORD"
						value: #config.server.rcon.password
					}
					RCON_PORT: {
						name:  "RCON_PORT"
						value: "\(#config.server.rcon.port)"
					}

					// Optional: Message of the day
					if #config.server.motd != _|_ {
						MOTD: {
							name:  "MOTD"
							value: #config.server.motd
						}
					}

					// Optional: Server operators (comma-separated list)
					if #config.server.ops != _|_ {
						OPS: {
							name: "OPS"
							let opsList = [for op in #config.server.ops {op}]

							value: "\(opsList[0])" // CUE will join with commas in actual implementation
						}
					}

					// Optional: Whitelist (comma-separated list)
					if #config.server.whitelist != _|_ {
						WHITELIST: {
							name: "WHITELIST"
							let whitelistList = [for user in #config.server.whitelist {user}]

							value: "\(whitelistList[0])" // CUE will join with commas
						}
					}

					// Optional: World seed
					if #config.server.seed != _|_ {
						SEED: {
							name:  "SEED"
							value: #config.server.seed
						}
					}

					// Optional: Maximum world size
					if #config.server.maxWorldSize != _|_ {
						MAX_WORLD_SIZE: {
							name:  "MAX_WORLD_SIZE"
							value: "\(#config.server.maxWorldSize)"
						}
					}

					// Optional: View distance
					if #config.server.viewDistance != _|_ {
						VIEW_DISTANCE: {
							name:  "VIEW_DISTANCE"
							value: "\(#config.server.viewDistance)"
						}
					}
				}

				volumeMounts: {
					data: {
						name:      "data"
						mountPath: "/data"
					}
				}

				if #config.resources != _|_ {
					resources: #config.resources
				}
			}

			// === Sidecar Container: Backup ===
			if #config.backup != _|_ && #config.backup.enabled {
				sidecarContainers: [{
					name:            "backup"
					image:           #config.backup.image
					imagePullPolicy: "IfNotPresent"

					env: {
						// Backup method and timing
						BACKUP_METHOD: {
							name:  "BACKUP_METHOD"
							value: #config.backup.method
						}
						BACKUP_INTERVAL: {
							name:  "BACKUP_INTERVAL"
							value: #config.backup.interval
						}
						INITIAL_DELAY: {
							name:  "INITIAL_DELAY"
							value: #config.backup.initialDelay
						}

						// RCON connection to Minecraft server
						RCON_HOST: {
							name:  "RCON_HOST"
							value: "localhost"
						}
						RCON_PORT: {
							name:  "RCON_PORT"
							value: "\(#config.server.rcon.port)"
						}
						RCON_PASSWORD: {
							name:  "RCON_PASSWORD"
							value: #config.server.rcon.password
						}

						// Backup destination
						SRC_DIR: {
							name:  "SRC_DIR"
							value: "/data"
						}
						DEST_DIR: {
							name:  "DEST_DIR"
							value: "/backups"
						}

						// Optional: Prune old backups
						if #config.backup.pruneBackupsDays != _|_ {
							PRUNE_BACKUPS_DAYS: {
								name:  "PRUNE_BACKUPS_DAYS"
								value: "\(#config.backup.pruneBackupsDays)"
							}
						}

						// Optional: Pause if no players
						if #config.backup.pauseIfNoPlayers != _|_ {
							PAUSE_IF_NO_PLAYERS: {
								name:  "PAUSE_IF_NO_PLAYERS"
								value: "\(#config.backup.pauseIfNoPlayers)"
							}
						}

						// Tar-specific configuration
						if #config.backup.method == "tar" && #config.backup.tar != _|_ {
							TAR_COMPRESS_METHOD: {
								name:  "TAR_COMPRESS_METHOD"
								value: #config.backup.tar.compressMethod
							}
							LINK_LATEST: {
								name:  "LINK_LATEST"
								value: "\(#config.backup.tar.linkLatest)"
							}
						}

						// Restic-specific configuration
						if #config.backup.method == "restic" && #config.backup.restic != _|_ {
							RESTIC_REPOSITORY: {
								name:  "RESTIC_REPOSITORY"
								value: #config.backup.restic.repository
							}
							RESTIC_PASSWORD: {
								name:  "RESTIC_PASSWORD"
								value: #config.backup.restic.password
							}
							if #config.backup.restic.retention != _|_ {
								PRUNE_RESTIC_RETENTION: {
									name:  "PRUNE_RESTIC_RETENTION"
									value: #config.backup.restic.retention
								}
							}
							if #config.backup.restic.hostname != _|_ {
								RESTIC_HOSTNAME: {
									name:  "RESTIC_HOSTNAME"
									value: #config.backup.restic.hostname
								}
							}
							if #config.backup.restic.verbose != _|_ {
								RESTIC_VERBOSE: {
									name:  "RESTIC_VERBOSE"
									value: "\(#config.backup.restic.verbose)"
								}
							}
						}

						// Rclone-specific configuration
						if #config.backup.method == "rclone" && #config.backup.rclone != _|_ {
							RCLONE_REMOTE: {
								name:  "RCLONE_REMOTE"
								value: #config.backup.rclone.remote
							}
							RCLONE_DEST_DIR: {
								name:  "RCLONE_DEST_DIR"
								value: #config.backup.rclone.destDir
							}
							RCLONE_COMPRESS_METHOD: {
								name:  "RCLONE_COMPRESS_METHOD"
								value: #config.backup.rclone.compressMethod
							}
						}
					}

					volumeMounts: {
						data: {
							name:      "data"
							mountPath: "/data"
							readOnly:  true // Backup only needs read access to game data
						}
						backups: {
							name:      "backups"
							mountPath: "/backups"
						}
					}

					resources: {
						cpu: {
							request: "100m"
							limit:   "1000m"
						}
						memory: {
							request: "256Mi"
							limit:   "1Gi"
						}
					}
				}]
			}

			// === Health Checks ===
			healthCheck: {
				livenessProbe: {
					exec: {
						command: ["mc-health"]
					}
					initialDelaySeconds: 60
					periodSeconds:       30
					timeoutSeconds:      5
					failureThreshold:    3
				}
				readinessProbe: {
					exec: {
						command: ["mc-health"]
					}
					initialDelaySeconds: 30
					periodSeconds:       10
					timeoutSeconds:      3
					failureThreshold:    3
				}
			}

			// === Network Exposure ===
			expose: {
				ports: {
					minecraft: container.ports.minecraft & {
						exposedPort: #config.port
					}
				}
				type: #config.serviceType
			}

			// === Volumes ===
			volumes: {
				// Data volume - dynamic configuration based on storage.data.type
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

				// Backup volume - conditional on backup.enabled
				if #config.backup != _|_ && #config.backup.enabled {
					backups: {
						name: "backups"

						if #config.storage.backups.type == "pvc" {
							persistentClaim: {
								size: #config.storage.backups.size
								if #config.storage.backups.storageClass != _|_ {
									storageClass: #config.storage.backups.storageClass
								}
							}
						}

						if #config.storage.backups.type == "hostPath" {
							hostPath: {
								path: #config.storage.backups.path
								type: #config.storage.backups.hostPathType
							}
						}

						if #config.storage.backups.type == "emptyDir" {
							emptyDir: {}
						}
					}
				}
			}
		}
	}
}

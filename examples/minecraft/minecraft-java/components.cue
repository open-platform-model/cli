// Components defines the Minecraft server workload with backup sidecar.
// Single stateful component with persistent data, optional backup, and health checks.
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
	//// Minecraft - Stateful Game Server with Backup Sidecar
	/////////////////////////////////////////////////////////////////

	server: {
		resources_workload.#Container
		resources_storage.#Volumes
		if #config.backup.enabled {
			traits_workload.#SidecarContainers
		}
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_workload.#UpdateStrategy
		traits_workload.#GracefulShutdown
		traits_network.#Expose
		traits_security.#SecurityContext

		metadata: labels: "core.opmodel.dev/workload-type": "stateful"

		spec: {
			// Single replica - Minecraft servers don't support horizontal scaling
			scaling: count: 1

			restartPolicy: "Always"

			// Recreate strategy - Minecraft cannot do rolling updates
			updateStrategy: type: "Recreate"

			// Graceful shutdown - allow time for world save
			gracefulShutdown: terminationGracePeriodSeconds: 60

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

			// === Main Container: Minecraft Server ===
			container: {
				name:  "server"
				image: #config.image

				ports: {
					minecraft: {
						targetPort: 25565
						protocol:   "TCP"
					}
					if #config.rcon.enabled {
						rcon: {
							name:       "rcon"
							targetPort: #config.rcon.port
							protocol:   "TCP"
						}
					}
					if #config.query.enabled {
						query: {
							name:       "query"
							targetPort: #config.query.port
							protocol:   "TCP"
						}
					}
				}

				env: {
					// Required: EULA acceptance
					EULA: {
						name:  "EULA"
						value: "\(#config.eula)"
					}

					// === Server Type ===
					// Derived from which type struct is present (matchN ensures exactly one)
					if #config.vanilla != _|_ {
						TYPE: {name: "TYPE", value: "VANILLA"}
					}
					if #config.paper != _|_ {
						TYPE: {name: "TYPE", value: "PAPER"}
					}
					if #config.forge != _|_ {
						TYPE: {name: "TYPE", value: "FORGE"}
					}
					if #config.fabric != _|_ {
						TYPE: {name: "TYPE", value: "FABRIC"}
					}
					if #config.spigot != _|_ {
						TYPE: {name: "TYPE", value: "SPIGOT"}
					}
					if #config.bukkit != _|_ {
						TYPE: {name: "TYPE", value: "BUKKIT"}
					}
					if #config.sponge != _|_ {
						TYPE: {name: "TYPE", value: "SPONGEVANILLA"}
					}
					if #config.purpur != _|_ {
						TYPE: {name: "TYPE", value: "PURPUR"}
					}
					if #config.magma != _|_ {
						TYPE: {name: "TYPE", value: "MAGMA"}
					}
					if #config.ftba != _|_ {
						TYPE: {name: "TYPE", value: "FTBA"}
					}
					if #config.autoCurseForge != _|_ {
						TYPE: {name: "TYPE", value: "AUTO_CURSEFORGE"}
					}

					// Game version
					VERSION: {
						name:  "VERSION"
						value: #config.version
					}

					// === Server Properties ===
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

					// RCON configuration
					if #config.rcon.enabled {
						ENABLE_RCON: {
							name:  "ENABLE_RCON"
							value: "true"
						}
						RCON_PASSWORD: {
							name: "RCON_PASSWORD"
							from: #config.rcon.password
						}
						RCON_PORT: {
							name:  "RCON_PORT"
							value: "\(#config.rcon.port)"
						}
					}

					// Optional: Message of the day
					if #config.server.motd != _|_ {
						MOTD: {
							name:  "MOTD"
							value: #config.server.motd
						}
					}

					// Optional: Server operators (comma-separated)
					if #config.server.ops != _|_ {
						OPS: {
							name:  "OPS"
							value: strings.Join(#config.server.ops, ",")
						}
					}

					// Optional: Blocklist (comma-separated)
					if #config.server.blocklist != _|_ {
						WHITELIST: {
							name:  "WHITELIST"
							value: strings.Join(#config.server.blocklist, ",")
						}
					}

					// Optional: World seed
					if #config.server.seed != _|_ {
						SEED: {
							name:  "SEED"
							value: #config.server.seed
						}
					}
					if #config.server.maxWorldSize != _|_ {
						MAX_WORLD_SIZE: {
							name:  "MAX_WORLD_SIZE"
							value: "\(#config.server.maxWorldSize)"
						}
					}
					VIEW_DISTANCE: {
						name:  "VIEW_DISTANCE"
						value: "\(#config.server.viewDistance)"
					}

					// === World Settings ===
					ALLOW_NETHER: {
						name:  "ALLOW_NETHER"
						value: "\(#config.server.allowNether)"
					}
					ANNOUNCE_PLAYER_ACHIEVEMENTS: {
						name:  "ANNOUNCE_PLAYER_ACHIEVEMENTS"
						value: "\(#config.server.announcePlayerAchievements)"
					}
					FORCE_GAMEMODE: {
						name:  "FORCE_GAMEMODE"
						value: "\(#config.server.forceGameMode)"
					}
					GENERATE_STRUCTURES: {
						name:  "GENERATE_STRUCTURES"
						value: "\(#config.server.generateStructures)"
					}
					HARDCORE: {
						name:  "HARDCORE"
						value: "\(#config.server.hardcore)"
					}
					MAX_BUILD_HEIGHT: {
						name:  "MAX_BUILD_HEIGHT"
						value: "\(#config.server.maxBuildHeight)"
					}
					MAX_TICK_TIME: {
						name:  "MAX_TICK_TIME"
						value: "\(#config.server.maxTickTime)"
					}
					SPAWN_ANIMALS: {
						name:  "SPAWN_ANIMALS"
						value: "\(#config.server.spawnAnimals)"
					}
					SPAWN_MONSTERS: {
						name:  "SPAWN_MONSTERS"
						value: "\(#config.server.spawnMonsters)"
					}
					SPAWN_NPCS: {
						name:  "SPAWN_NPCS"
						value: "\(#config.server.spawnNPCs)"
					}
					SPAWN_PROTECTION: {
						name:  "SPAWN_PROTECTION"
						value: "\(#config.server.spawnProtection)"
					}
					LEVEL_TYPE: {
						name:  "LEVEL_TYPE"
						value: #config.server.levelType
					}
					LEVEL: {
						name:  "LEVEL"
						value: #config.server.worldSaveName
					}
					ONLINE_MODE: {
						name:  "ONLINE_MODE"
						value: "\(#config.server.onlineMode)"
					}
					ENFORCE_SECURE_PROFILE: {
						name:  "ENFORCE_SECURE_PROFILE"
						value: "\(#config.server.enforceSecureProfile)"
					}
					OVERRIDE_SERVER_PROPERTIES: {
						name:  "OVERRIDE_SERVER_PROPERTIES"
						value: "\(#config.server.overrideServerProperties)"
					}

					// === JVM Configuration ===
					MEMORY: {
						name:  "MEMORY"
						value: #config.jvm.memory
					}
					if #config.jvm.opts != _|_ {
						JVM_OPTS: {
							name:  "JVM_OPTS"
							value: #config.jvm.opts
						}
					}
					if #config.jvm.xxOpts != _|_ {
						JVM_XX_OPTS: {
							name:  "JVM_XX_OPTS"
							value: #config.jvm.xxOpts
						}
					}

					// === Type-Specific Configuration ===
					if #config.forge != _|_ {
						FORGE_VERSION: {
							name:  "FORGE_VERSION"
							value: #config.forge.version
						}
						if #config.forge.installerUrl != _|_ {
							FORGE_INSTALLER_URL: {
								name:  "FORGE_INSTALLER_URL"
								value: #config.forge.installerUrl
							}
						}
						if #config.forge.mods != _|_ {
							if #config.forge.mods.urls != _|_ {
								MODS: {
									name:  "MODS"
									value: strings.Join(#config.forge.mods.urls, ",")
								}
							}
							if #config.forge.mods.modrinth != _|_ {
								MODRINTH_PROJECTS: {
									name:  "MODRINTH_PROJECTS"
									value: strings.Join(#config.forge.mods.modrinth.projects, ",")
								}
								if #config.forge.mods.modrinth.downloadDependencies != _|_ {
									MODRINTH_DOWNLOAD_DEPENDENCIES: {
										name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
										value: #config.forge.mods.modrinth.downloadDependencies
									}
								}
								if #config.forge.mods.modrinth.allowedVersionType != _|_ {
									MODRINTH_ALLOWED_VERSION_TYPE: {
										name:  "MODRINTH_ALLOWED_VERSION_TYPE"
										value: #config.forge.mods.modrinth.allowedVersionType
									}
								}
							}
							if #config.forge.mods.modpackUrl != _|_ {
								MODPACK: {
									name:  "MODPACK"
									value: #config.forge.mods.modpackUrl
								}
							}
							REMOVE_OLD_MODS: {
								name:  "REMOVE_OLD_MODS"
								value: "\(#config.forge.mods.removeOldMods)"
							}
						}
					}
					if #config.fabric != _|_ {
						FABRIC_LOADER_VERSION: {
							name:  "FABRIC_LOADER_VERSION"
							value: #config.fabric.loaderVersion
						}
						if #config.fabric.installerUrl != _|_ {
							FABRIC_INSTALLER_URL: {
								name:  "FABRIC_INSTALLER_URL"
								value: #config.fabric.installerUrl
							}
						}
						if #config.fabric.mods != _|_ {
							if #config.fabric.mods.urls != _|_ {
								MODS: {
									name:  "MODS"
									value: strings.Join(#config.fabric.mods.urls, ",")
								}
							}
							if #config.fabric.mods.modrinth != _|_ {
								MODRINTH_PROJECTS: {
									name:  "MODRINTH_PROJECTS"
									value: strings.Join(#config.fabric.mods.modrinth.projects, ",")
								}
								if #config.fabric.mods.modrinth.downloadDependencies != _|_ {
									MODRINTH_DOWNLOAD_DEPENDENCIES: {
										name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
										value: #config.fabric.mods.modrinth.downloadDependencies
									}
								}
								if #config.fabric.mods.modrinth.allowedVersionType != _|_ {
									MODRINTH_ALLOWED_VERSION_TYPE: {
										name:  "MODRINTH_ALLOWED_VERSION_TYPE"
										value: #config.fabric.mods.modrinth.allowedVersionType
									}
								}
							}
							if #config.fabric.mods.modpackUrl != _|_ {
								MODPACK: {
									name:  "MODPACK"
									value: #config.fabric.mods.modpackUrl
								}
							}
							REMOVE_OLD_MODS: {
								name:  "REMOVE_OLD_MODS"
								value: "\(#config.fabric.mods.removeOldMods)"
							}
						}
					}
					if #config.paper != _|_ {
						if #config.paper.downloadUrl != _|_ {
							PAPER_DOWNLOAD_URL: {
								name:  "PAPER_DOWNLOAD_URL"
								value: #config.paper.downloadUrl
							}
						}
						if #config.paper.plugins != _|_ {
							if #config.paper.plugins.urls != _|_ {
								PLUGINS: {
									name:  "PLUGINS"
									value: strings.Join(#config.paper.plugins.urls, ",")
								}
							}
							if #config.paper.plugins.spigetResources != _|_ {
								let _spigetStrings = [for r in #config.paper.plugins.spigetResources {"\(r)"}]
								SPIGET_RESOURCES: {
									name:  "SPIGET_RESOURCES"
									value: strings.Join(_spigetStrings, ",")
								}
							}
							if #config.paper.plugins.modrinth != _|_ {
								MODRINTH_PROJECTS: {
									name:  "MODRINTH_PROJECTS"
									value: strings.Join(#config.paper.plugins.modrinth.projects, ",")
								}
								if #config.paper.plugins.modrinth.downloadDependencies != _|_ {
									MODRINTH_DOWNLOAD_DEPENDENCIES: {
										name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
										value: #config.paper.plugins.modrinth.downloadDependencies
									}
								}
								if #config.paper.plugins.modrinth.allowedVersionType != _|_ {
									MODRINTH_ALLOWED_VERSION_TYPE: {
										name:  "MODRINTH_ALLOWED_VERSION_TYPE"
										value: #config.paper.plugins.modrinth.allowedVersionType
									}
								}
							}
							REMOVE_OLD_MODS: {
								name:  "REMOVE_OLD_MODS"
								value: "\(#config.paper.plugins.removeOldMods)"
							}
						}
					}
					if #config.spigot != _|_ {
						if #config.spigot.downloadUrl != _|_ {
							SPIGOT_DOWNLOAD_URL: {
								name:  "SPIGOT_DOWNLOAD_URL"
								value: #config.spigot.downloadUrl
							}
						}
						if #config.spigot.plugins != _|_ {
							if #config.spigot.plugins.urls != _|_ {
								PLUGINS: {
									name:  "PLUGINS"
									value: strings.Join(#config.spigot.plugins.urls, ",")
								}
							}
							if #config.spigot.plugins.spigetResources != _|_ {
								let _spigetStrings = [for r in #config.spigot.plugins.spigetResources {"\(r)"}]
								SPIGET_RESOURCES: {
									name:  "SPIGET_RESOURCES"
									value: strings.Join(_spigetStrings, ",")
								}
							}
							if #config.spigot.plugins.modrinth != _|_ {
								MODRINTH_PROJECTS: {
									name:  "MODRINTH_PROJECTS"
									value: strings.Join(#config.spigot.plugins.modrinth.projects, ",")
								}
								if #config.spigot.plugins.modrinth.downloadDependencies != _|_ {
									MODRINTH_DOWNLOAD_DEPENDENCIES: {
										name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
										value: #config.spigot.plugins.modrinth.downloadDependencies
									}
								}
								if #config.spigot.plugins.modrinth.allowedVersionType != _|_ {
									MODRINTH_ALLOWED_VERSION_TYPE: {
										name:  "MODRINTH_ALLOWED_VERSION_TYPE"
										value: #config.spigot.plugins.modrinth.allowedVersionType
									}
								}
							}
							REMOVE_OLD_MODS: {
								name:  "REMOVE_OLD_MODS"
								value: "\(#config.spigot.plugins.removeOldMods)"
							}
						}
					}
					if #config.bukkit != _|_ {
						if #config.bukkit.downloadUrl != _|_ {
							BUKKIT_DOWNLOAD_URL: {
								name:  "BUKKIT_DOWNLOAD_URL"
								value: #config.bukkit.downloadUrl
							}
						}
						if #config.bukkit.plugins != _|_ {
							if #config.bukkit.plugins.urls != _|_ {
								PLUGINS: {
									name:  "PLUGINS"
									value: strings.Join(#config.bukkit.plugins.urls, ",")
								}
							}
							if #config.bukkit.plugins.spigetResources != _|_ {
								let _spigetStrings = [for r in #config.bukkit.plugins.spigetResources {"\(r)"}]
								SPIGET_RESOURCES: {
									name:  "SPIGET_RESOURCES"
									value: strings.Join(_spigetStrings, ",")
								}
							}
							if #config.bukkit.plugins.modrinth != _|_ {
								MODRINTH_PROJECTS: {
									name:  "MODRINTH_PROJECTS"
									value: strings.Join(#config.bukkit.plugins.modrinth.projects, ",")
								}
								if #config.bukkit.plugins.modrinth.downloadDependencies != _|_ {
									MODRINTH_DOWNLOAD_DEPENDENCIES: {
										name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
										value: #config.bukkit.plugins.modrinth.downloadDependencies
									}
								}
								if #config.bukkit.plugins.modrinth.allowedVersionType != _|_ {
									MODRINTH_ALLOWED_VERSION_TYPE: {
										name:  "MODRINTH_ALLOWED_VERSION_TYPE"
										value: #config.bukkit.plugins.modrinth.allowedVersionType
									}
								}
							}
							REMOVE_OLD_MODS: {
								name:  "REMOVE_OLD_MODS"
								value: "\(#config.bukkit.plugins.removeOldMods)"
							}
						}
					}
					if #config.sponge != _|_ {
						SPONGEVERSION: {
							name:  "SPONGEVERSION"
							value: #config.sponge.version
						}
					}
					if #config.purpur != _|_ {
						if #config.purpur.plugins != _|_ {
							if #config.purpur.plugins.urls != _|_ {
								PLUGINS: {
									name:  "PLUGINS"
									value: strings.Join(#config.purpur.plugins.urls, ",")
								}
							}
							if #config.purpur.plugins.spigetResources != _|_ {
								let _spigetStrings = [for r in #config.purpur.plugins.spigetResources {"\(r)"}]
								SPIGET_RESOURCES: {
									name:  "SPIGET_RESOURCES"
									value: strings.Join(_spigetStrings, ",")
								}
							}
							if #config.purpur.plugins.modrinth != _|_ {
								MODRINTH_PROJECTS: {
									name:  "MODRINTH_PROJECTS"
									value: strings.Join(#config.purpur.plugins.modrinth.projects, ",")
								}
								if #config.purpur.plugins.modrinth.downloadDependencies != _|_ {
									MODRINTH_DOWNLOAD_DEPENDENCIES: {
										name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
										value: #config.purpur.plugins.modrinth.downloadDependencies
									}
								}
								if #config.purpur.plugins.modrinth.allowedVersionType != _|_ {
									MODRINTH_ALLOWED_VERSION_TYPE: {
										name:  "MODRINTH_ALLOWED_VERSION_TYPE"
										value: #config.purpur.plugins.modrinth.allowedVersionType
									}
								}
							}
							REMOVE_OLD_MODS: {
								name:  "REMOVE_OLD_MODS"
								value: "\(#config.purpur.plugins.removeOldMods)"
							}
						}
					}
					if #config.magma != _|_ {
						if #config.magma.mods != _|_ {
							if #config.magma.mods.urls != _|_ {
								MODS: {
									name:  "MODS"
									value: strings.Join(#config.magma.mods.urls, ",")
								}
							}
							if #config.magma.mods.modrinth != _|_ {
								MODRINTH_PROJECTS: {
									name:  "MODRINTH_PROJECTS"
									value: strings.Join(#config.magma.mods.modrinth.projects, ",")
								}
								if #config.magma.mods.modrinth.downloadDependencies != _|_ {
									MODRINTH_DOWNLOAD_DEPENDENCIES: {
										name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
										value: #config.magma.mods.modrinth.downloadDependencies
									}
								}
								if #config.magma.mods.modrinth.allowedVersionType != _|_ {
									MODRINTH_ALLOWED_VERSION_TYPE: {
										name:  "MODRINTH_ALLOWED_VERSION_TYPE"
										value: #config.magma.mods.modrinth.allowedVersionType
									}
								}
							}
							if #config.magma.mods.modpackUrl != _|_ {
								MODPACK: {
									name:  "MODPACK"
									value: #config.magma.mods.modpackUrl
								}
							}
							REMOVE_OLD_MODS: {
								name:  "REMOVE_OLD_MODS"
								value: "\(#config.magma.mods.removeOldMods)"
							}
						}
						if #config.magma.plugins != _|_ {
							if #config.magma.plugins.urls != _|_ {
								PLUGINS: {
									name:  "PLUGINS"
									value: strings.Join(#config.magma.plugins.urls, ",")
								}
							}
							if #config.magma.plugins.spigetResources != _|_ {
								let _spigetStrings = [for r in #config.magma.plugins.spigetResources {"\(r)"}]
								SPIGET_RESOURCES: {
									name:  "SPIGET_RESOURCES"
									value: strings.Join(_spigetStrings, ",")
								}
							}
						}
					}
					if #config.ftba != _|_ {
						if #config.ftba.mods != _|_ {
							if #config.ftba.mods.urls != _|_ {
								MODS: {
									name:  "MODS"
									value: strings.Join(#config.ftba.mods.urls, ",")
								}
							}
							if #config.ftba.mods.modrinth != _|_ {
								MODRINTH_PROJECTS: {
									name:  "MODRINTH_PROJECTS"
									value: strings.Join(#config.ftba.mods.modrinth.projects, ",")
								}
								if #config.ftba.mods.modrinth.downloadDependencies != _|_ {
									MODRINTH_DOWNLOAD_DEPENDENCIES: {
										name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
										value: #config.ftba.mods.modrinth.downloadDependencies
									}
								}
								if #config.ftba.mods.modrinth.allowedVersionType != _|_ {
									MODRINTH_ALLOWED_VERSION_TYPE: {
										name:  "MODRINTH_ALLOWED_VERSION_TYPE"
										value: #config.ftba.mods.modrinth.allowedVersionType
									}
								}
							}
							if #config.ftba.mods.modpackUrl != _|_ {
								MODPACK: {
									name:  "MODPACK"
									value: #config.ftba.mods.modpackUrl
								}
							}
							REMOVE_OLD_MODS: {
								name:  "REMOVE_OLD_MODS"
								value: "\(#config.ftba.mods.removeOldMods)"
							}
						}
					}
					if #config.autoCurseForge != _|_ {
						CF_API_KEY: {
							name: "CF_API_KEY"
							from: #config.autoCurseForge.apiKey
						}
						if #config.autoCurseForge.pageUrl != _|_ {
							CF_PAGE_URL: {
								name:  "CF_PAGE_URL"
								value: #config.autoCurseForge.pageUrl
							}
						}
						if #config.autoCurseForge.slug != _|_ {
							CF_SLUG: {
								name:  "CF_SLUG"
								value: #config.autoCurseForge.slug
							}
						}
						if #config.autoCurseForge.fileId != _|_ {
							CF_FILE_ID: {
								name:  "CF_FILE_ID"
								value: #config.autoCurseForge.fileId
							}
						}
						if #config.autoCurseForge.filenameMatcher != _|_ {
							CF_FILENAME_MATCHER: {
								name:  "CF_FILENAME_MATCHER"
								value: #config.autoCurseForge.filenameMatcher
							}
						}
						if #config.autoCurseForge.forceSynchronize != _|_ {
							CF_FORCE_SYNCHRONIZE: {
								name:  "CF_FORCE_SYNCHRONIZE"
								value: "\(#config.autoCurseForge.forceSynchronize)"
							}
						}
						if #config.autoCurseForge.parallelDownloads != _|_ {
							CF_PARALLEL_DOWNLOADS: {
								name:  "CF_PARALLEL_DOWNLOADS"
								value: "\(#config.autoCurseForge.parallelDownloads)"
							}
						}
					}

					// === Resource Packs (server.properties) ===
					if #config.server.resourcePackUrl != _|_ {
						RESOURCE_PACK: {
							name:  "RESOURCE_PACK"
							value: #config.server.resourcePackUrl
						}
					}
					if #config.server.resourcePackSha != _|_ {
						RESOURCE_PACK_SHA1: {
							name:  "RESOURCE_PACK_SHA1"
							value: #config.server.resourcePackSha
						}
					}
					if #config.server.resourcePackEnforce != _|_ {
						RESOURCE_PACK_ENFORCE: {
							name:  "RESOURCE_PACK_ENFORCE"
							value: "\(#config.server.resourcePackEnforce)"
						}
					}

					// === VanillaTweaks ===
					if #config.server.vanillaTweaksShareCodes != _|_ {
						VANILLATWEAKS_SHARECODE: {
							name:  "VANILLATWEAKS_SHARECODE"
							value: strings.Join(#config.server.vanillaTweaksShareCodes, ",")
						}
					}

					// === World Data ===
					if #config.downloadWorldUrl != _|_ {
						WORLD: {
							name:  "WORLD"
							value: #config.downloadWorldUrl
						}
					}

					// === Aikar's GC Flags ===
					if #config.jvm.useAikarFlags {
						USE_AIKAR_FLAGS: {
							name:  "USE_AIKAR_FLAGS"
							value: "true"
						}
					}

					// === Query Port ===
					if #config.query != _|_ {
						if #config.query.enabled {
							ENABLE_QUERY: {
								name:  "ENABLE_QUERY"
								value: "true"
							}
							QUERY_PORT: {
								name:  "QUERY_PORT"
								value: "\(#config.query.port)"
							}
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

				// === Health Checks ===
				// Startup probe: allow up to 10 min for world/chunk loading
				startupProbe: {
					exec: command: ["mc-monitor", "status", "--port", "\(#config.port)"]
					periodSeconds:    10
					timeoutSeconds:   5
					failureThreshold: 60
				}
				livenessProbe: {
					exec: command: ["mc-monitor", "status", "--port", "\(#config.port)"]
					periodSeconds:    30
					timeoutSeconds:   5
					failureThreshold: 3
				}
				readinessProbe: {
					exec: command: ["mc-monitor", "status", "--port", "\(#config.port)"]
					periodSeconds:    10
					timeoutSeconds:   3
					failureThreshold: 3
				}
			}

			// === Sidecar Container: Backup ===
			if #config.backup.enabled {
				sidecarContainers: [{
					name:  "backup"
					image: #config.backup.image

					env: {
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
						RCON_HOST: {
							name:  "RCON_HOST"
							value: "localhost"
						}
						RCON_PORT: {
							name:  "RCON_PORT"
							value: "\(#config.rcon.port)"
						}
						RCON_PASSWORD: {
							name: "RCON_PASSWORD"
							from: #config.rcon.password
						}
						SRC_DIR: {
							name:  "SRC_DIR"
							value: "/data"
						}
						DEST_DIR: {
							name:  "DEST_DIR"
							value: "/backups"
						}
						if #config.backup.pruneBackupsDays != _|_ {
							PRUNE_BACKUPS_DAYS: {
								name:  "PRUNE_BACKUPS_DAYS"
								value: "\(#config.backup.pruneBackupsDays)"
							}
						}
						PAUSE_IF_NO_PLAYERS: {
							name:  "PAUSE_IF_NO_PLAYERS"
							value: "\(#config.backup.pauseIfNoPlayers)"
						}
						if #config.backup.method == "tar" {
							if #config.backup.tar != _|_ {
								TAR_COMPRESS_METHOD: {
									name:  "TAR_COMPRESS_METHOD"
									value: #config.backup.tar.compressMethod
								}
								LINK_LATEST: {
									name:  "LINK_LATEST"
									value: "\(#config.backup.tar.linkLatest)"
								}
							}
						}
						if #config.backup.method == "restic" {
							if #config.backup.restic != _|_ {
								RESTIC_REPOSITORY: {
									name:  "RESTIC_REPOSITORY"
									value: #config.backup.restic.repository
								}
								RESTIC_PASSWORD: {
									name: "RESTIC_PASSWORD"
									from: #config.backup.restic.password
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
						}
						if #config.backup.method == "rclone" {
							if #config.backup.rclone != _|_ {
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
					}

					volumeMounts: {
						data: volumes.data & {
							mountPath: "/data"
							readOnly:  true
						}
						backups: volumes.backups & {
							mountPath: "/backups"
						}
					}

					resources: {
						requests: {
							cpu:    "100m"
							memory: "256Mi"
						}
						limits: {
							cpu:    "1000m"
							memory: "1Gi"
						}
					}
				}]
			}

			// === Network Exposure ===
			expose: {
				ports: minecraft: {
					targetPort:  25565
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
				if #config.backup.enabled {
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

// Components defines the Minecraft Java server fleet workload.
//
// Dynamic component generation:
//   - One `server-{name}` component per entry in #config.servers
//     (StatefulSet + Service, identical structure to mc_java)
//   - One `router` component (stateless mc-router Deployment + LoadBalancer Service)
//     with --mapping args auto-built from the servers map
//   - One `rbac` component granting mc-router K8s service discovery permissions
//
// Router --mapping arg format:
//   -mapping={name}.{domain}={releaseName}-server-{name}.{namespace}.svc:{port}
package mc_java_fleet

import (
	"list"
	"strings"

	resources_workload "opmodel.dev/resources/workload@v1"
	resources_storage "opmodel.dev/resources/storage@v1"
	resources_security "opmodel.dev/resources/security@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
	traits_security "opmodel.dev/traits/security@v1"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	// Pre-computed shared bindings to avoid repeating #config.xxx in every comprehension
	// and ensure string interpolation has concrete values.
	let _domain = #config.domain
	let _relName = #config.releaseName
	let _ns = #config.namespace

	// ── Dynamic Minecraft server components ──────────────────────────────────────
	// One StatefulSet + Service per entry in #config.servers.
	// Component name: server-{name}  →  K8s Service: {releaseName}-server-{name}
	for _srvName, _srvCfg in #config.servers {
		let _c = _srvCfg

		"server-\(_srvName)": {
			resources_workload.#Container
			resources_storage.#Volumes
			if _c.seed.url != _|_ {
				traits_workload.#InitContainers
			}
			if _c.backup.enabled || _c.monitor.enabled {
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
				// Single replica — Minecraft servers don't support horizontal scaling
				scaling: count: 1

				restartPolicy: "Always"

				// Recreate strategy — Minecraft cannot do rolling updates
				updateStrategy: type: "Recreate"

				// Graceful shutdown — allow time for world save
				gracefulShutdown: terminationGracePeriodSeconds: 60

				// === World Seed Init Container ===
				// Only injected when seed.url is set.
				// Downloads a tar.gz and extracts only world directories (level.dat-bearing)
				// into /data. Writes a sentinel on success; subsequent pod restarts skip it.
				if _c.seed.url != _|_ {
					let _seedUrl = _c.seed.url
					initContainers: [{
						name:  "world-seed"
						image: _c.seed.image
						command: ["/bin/sh", "-c"]
						args: ["""
							set -e
							SENTINEL="/data/.opm-world-seed-done"
							if [ -f "$SENTINEL" ]; then
							  echo "World seed already applied (sentinel exists), skipping."
							  exit 0
							fi
							TMPDIR="/data/._seed_tmp"
							mkdir -p "$TMPDIR"
							echo "Downloading world archive from \(_seedUrl) ..."
							curl -fL "\(_seedUrl)" | tar -xz -C "$TMPDIR"
							echo "Scanning archive for world directories (level.dat detection)..."
							find "$TMPDIR" -maxdepth 2 -name "level.dat" | while read -r leveldat; do
							  worlddir=$(dirname "$leveldat")
							  worldname=$(basename "$worlddir")
							  if [ -d "/data/$worldname" ]; then
							    echo "Skip $worldname: already exists in /data"
							  else
							    echo "Seeding world: $worldname"
							    cp -a "$worlddir" /data/
							  fi
							done
							rm -rf "$TMPDIR"
							echo "$(date -Iseconds) opm-world-seed complete" > "$SENTINEL"
							echo "World seed complete."
							"""]
						volumeMounts: data: {
							name:      "data"
							mountPath: "/data"
						}
					}]
				}

				// === Security Context ===
				if _c.securityContext != _|_ {
					securityContext: _c.securityContext
				}
				if _c.securityContext == _|_ {
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
					image: _c.image

					ports: {
						minecraft: {
							targetPort: 25565
							protocol:   "TCP"
						}
						if _c.rcon.enabled {
							rcon: {
								name:       "rcon"
								targetPort: _c.rcon.port
								protocol:   "TCP"
							}
						}
						if _c.query.enabled {
							query: {
								name:       "query"
								targetPort: _c.query.port
								protocol:   "TCP"
							}
						}
					}

					env: {
						EULA: {
							name:  "EULA"
							value: "\(_c.eula)"
						}

						// === Server Type ===
						if _c.vanilla != _|_ {
							TYPE: {name: "TYPE", value: "VANILLA"}
						}
						if _c.paper != _|_ {
							TYPE: {name: "TYPE", value: "PAPER"}
						}
						if _c.forge != _|_ {
							TYPE: {name: "TYPE", value: "FORGE"}
						}
						if _c.fabric != _|_ {
							TYPE: {name: "TYPE", value: "FABRIC"}
						}
						if _c.spigot != _|_ {
							TYPE: {name: "TYPE", value: "SPIGOT"}
						}
						if _c.bukkit != _|_ {
							TYPE: {name: "TYPE", value: "BUKKIT"}
						}
						if _c.sponge != _|_ {
							TYPE: {name: "TYPE", value: "SPONGEVANILLA"}
						}
						if _c.purpur != _|_ {
							TYPE: {name: "TYPE", value: "PURPUR"}
						}
						if _c.magma != _|_ {
							TYPE: {name: "TYPE", value: "MAGMA"}
						}
						if _c.ftba != _|_ {
							TYPE: {name: "TYPE", value: "FTBA"}
						}
						if _c.autoCurseForge != _|_ {
							TYPE: {name: "TYPE", value: "AUTO_CURSEFORGE"}
						}
						if _c.modrinth != _|_ {
							TYPE: {name: "TYPE", value: "MODRINTH"}
						}

						VERSION: {
							name:  "VERSION"
							value: _c.version
						}

						// === Server Properties ===
						MAX_PLAYERS: {
							name:  "MAX_PLAYERS"
							value: "\(_c.server.maxPlayers)"
						}
						DIFFICULTY: {
							name:  "DIFFICULTY"
							value: _c.server.difficulty
						}
						MODE: {
							name:  "MODE"
							value: _c.server.mode
						}
						PVP: {
							name:  "PVP"
							value: "\(_c.server.pvp)"
						}
						ENABLE_COMMAND_BLOCK: {
							name:  "ENABLE_COMMAND_BLOCK"
							value: "\(_c.server.enableCommandBlock)"
						}

						if _c.rcon.enabled {
							ENABLE_RCON: {
								name:  "ENABLE_RCON"
								value: "true"
							}
							// Shared RCON password injected from module-level config
							RCON_PASSWORD: {
								name: "RCON_PASSWORD"
								from: #config.rconPassword
							}
							RCON_PORT: {
								name:  "RCON_PORT"
								value: "\(_c.rcon.port)"
							}
						}

						if _c.server.motd != _|_ {
							MOTD: {
								name:  "MOTD"
								value: _c.server.motd
							}
						}
						if _c.server.ops != _|_ {
							OPS: {
								name:  "OPS"
								value: strings.Join(_c.server.ops, ",")
							}
						}
						if _c.server.blocklist != _|_ {
							WHITELIST: {
								name:  "WHITELIST"
								value: strings.Join(_c.server.blocklist, ",")
							}
						}
						if _c.server.seed != _|_ {
							SEED: {
								name:  "SEED"
								value: _c.server.seed
							}
						}
						if _c.server.maxWorldSize != _|_ {
							MAX_WORLD_SIZE: {
								name:  "MAX_WORLD_SIZE"
								value: "\(_c.server.maxWorldSize)"
							}
						}
						VIEW_DISTANCE: {
							name:  "VIEW_DISTANCE"
							value: "\(_c.server.viewDistance)"
						}
						ALLOW_NETHER: {
							name:  "ALLOW_NETHER"
							value: "\(_c.server.allowNether)"
						}
						ALLOW_FLIGHT: {
							name:  "ALLOW_FLIGHT"
							value: "\(_c.server.allowFlight)"
						}
						ENABLE_ROLLING_LOGS: {
							name:  "ENABLE_ROLLING_LOGS"
							value: "\(_c.server.enableRollingLogs)"
						}
						if _c.server.serverName != _|_ {
							SERVER_NAME: {
								name:  "SERVER_NAME"
								value: _c.server.serverName
							}
						}
						if _c.server.tz != _|_ {
							TZ: {
								name:  "TZ"
								value: _c.server.tz
							}
						}
						ANNOUNCE_PLAYER_ACHIEVEMENTS: {
							name:  "ANNOUNCE_PLAYER_ACHIEVEMENTS"
							value: "\(_c.server.announcePlayerAchievements)"
						}
						FORCE_GAMEMODE: {
							name:  "FORCE_GAMEMODE"
							value: "\(_c.server.forceGameMode)"
						}
						GENERATE_STRUCTURES: {
							name:  "GENERATE_STRUCTURES"
							value: "\(_c.server.generateStructures)"
						}
						HARDCORE: {
							name:  "HARDCORE"
							value: "\(_c.server.hardcore)"
						}
						MAX_BUILD_HEIGHT: {
							name:  "MAX_BUILD_HEIGHT"
							value: "\(_c.server.maxBuildHeight)"
						}
						MAX_TICK_TIME: {
							name:  "MAX_TICK_TIME"
							value: "\(_c.server.maxTickTime)"
						}
						SPAWN_ANIMALS: {
							name:  "SPAWN_ANIMALS"
							value: "\(_c.server.spawnAnimals)"
						}
						SPAWN_MONSTERS: {
							name:  "SPAWN_MONSTERS"
							value: "\(_c.server.spawnMonsters)"
						}
						SPAWN_NPCS: {
							name:  "SPAWN_NPCS"
							value: "\(_c.server.spawnNPCs)"
						}
						SPAWN_PROTECTION: {
							name:  "SPAWN_PROTECTION"
							value: "\(_c.server.spawnProtection)"
						}
						LEVEL_TYPE: {
							name:  "LEVEL_TYPE"
							value: _c.server.levelType
						}
						LEVEL: {
							name:  "LEVEL"
							value: _c.server.worldSaveName
						}
						ONLINE_MODE: {
							name:  "ONLINE_MODE"
							value: "\(_c.server.onlineMode)"
						}
						ENFORCE_SECURE_PROFILE: {
							name:  "ENFORCE_SECURE_PROFILE"
							value: "\(_c.server.enforceSecureProfile)"
						}
						OVERRIDE_SERVER_PROPERTIES: {
							name:  "OVERRIDE_SERVER_PROPERTIES"
							value: "\(_c.server.overrideServerProperties)"
						}

						// === JVM ===
						// When maxMemory is set it becomes MEMORY; initMemory becomes INIT_MEMORY.
						// When only memory is set it becomes MEMORY (single heap).
						if _c.jvm.maxMemory != _|_ {
							MEMORY: {
								name:  "MEMORY"
								value: _c.jvm.maxMemory
							}
							if _c.jvm.initMemory != _|_ {
								INIT_MEMORY: {
									name:  "INIT_MEMORY"
									value: _c.jvm.initMemory
								}
							}
						}
						if _c.jvm.maxMemory == _|_ {
							MEMORY: {
								name:  "MEMORY"
								value: _c.jvm.memory
							}
						}
						if _c.jvm.opts != _|_ {
							JVM_OPTS: {
								name:  "JVM_OPTS"
								value: _c.jvm.opts
							}
						}
						if _c.jvm.xxOpts != _|_ {
							JVM_XX_OPTS: {
								name:  "JVM_XX_OPTS"
								value: _c.jvm.xxOpts
							}
						}
						if _c.jvm.useAikarFlags {
							USE_AIKAR_FLAGS: {
								name:  "USE_AIKAR_FLAGS"
								value: "true"
							}
						}

						// === Type-Specific ===
						if _c.forge != _|_ {
							FORGE_VERSION: {
								name:  "FORGE_VERSION"
								value: _c.forge.version
							}
							if _c.forge.installerUrl != _|_ {
								FORGE_INSTALLER_URL: {
									name:  "FORGE_INSTALLER_URL"
									value: _c.forge.installerUrl
								}
							}
							if _c.forge.mods != _|_ {
								if _c.forge.mods.urls != _|_ {
									MODS: {
										name:  "MODS"
										value: strings.Join(_c.forge.mods.urls, ",")
									}
								}
								if _c.forge.mods.modrinth != _|_ {
									MODRINTH_PROJECTS: {
										name:  "MODRINTH_PROJECTS"
										value: strings.Join(_c.forge.mods.modrinth.projects, ",")
									}
									if _c.forge.mods.modrinth.downloadDependencies != _|_ {
										MODRINTH_DOWNLOAD_DEPENDENCIES: {
											name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
											value: _c.forge.mods.modrinth.downloadDependencies
										}
									}
									if _c.forge.mods.modrinth.allowedVersionType != _|_ {
										MODRINTH_ALLOWED_VERSION_TYPE: {
											name:  "MODRINTH_ALLOWED_VERSION_TYPE"
											value: _c.forge.mods.modrinth.allowedVersionType
										}
									}
								}
								if _c.forge.mods.modpackUrl != _|_ {
									MODPACK: {
										name:  "MODPACK"
										value: _c.forge.mods.modpackUrl
									}
								}
								REMOVE_OLD_MODS: {
									name:  "REMOVE_OLD_MODS"
									value: "\(_c.forge.mods.removeOldMods)"
								}
							}
						}
						if _c.fabric != _|_ {
							FABRIC_LOADER_VERSION: {
								name:  "FABRIC_LOADER_VERSION"
								value: _c.fabric.loaderVersion
							}
							if _c.fabric.installerUrl != _|_ {
								FABRIC_INSTALLER_URL: {
									name:  "FABRIC_INSTALLER_URL"
									value: _c.fabric.installerUrl
								}
							}
							if _c.fabric.mods != _|_ {
								if _c.fabric.mods.urls != _|_ {
									MODS: {
										name:  "MODS"
										value: strings.Join(_c.fabric.mods.urls, ",")
									}
								}
								if _c.fabric.mods.modrinth != _|_ {
									MODRINTH_PROJECTS: {
										name:  "MODRINTH_PROJECTS"
										value: strings.Join(_c.fabric.mods.modrinth.projects, ",")
									}
									if _c.fabric.mods.modrinth.downloadDependencies != _|_ {
										MODRINTH_DOWNLOAD_DEPENDENCIES: {
											name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
											value: _c.fabric.mods.modrinth.downloadDependencies
										}
									}
									if _c.fabric.mods.modrinth.allowedVersionType != _|_ {
										MODRINTH_ALLOWED_VERSION_TYPE: {
											name:  "MODRINTH_ALLOWED_VERSION_TYPE"
											value: _c.fabric.mods.modrinth.allowedVersionType
										}
									}
								}
								if _c.fabric.mods.modpackUrl != _|_ {
									MODPACK: {
										name:  "MODPACK"
										value: _c.fabric.mods.modpackUrl
									}
								}
								REMOVE_OLD_MODS: {
									name:  "REMOVE_OLD_MODS"
									value: "\(_c.fabric.mods.removeOldMods)"
								}
							}
						}
						if _c.paper != _|_ {
							if _c.paper.downloadUrl != _|_ {
								PAPER_DOWNLOAD_URL: {
									name:  "PAPER_DOWNLOAD_URL"
									value: _c.paper.downloadUrl
								}
							}
							if _c.paper.plugins != _|_ {
								if _c.paper.plugins.urls != _|_ {
									PLUGINS: {
										name:  "PLUGINS"
										value: strings.Join(_c.paper.plugins.urls, ",")
									}
								}
								if _c.paper.plugins.spigetResources != _|_ {
									let _spigetStrings = [for r in _c.paper.plugins.spigetResources {"\(r)"}]
									SPIGET_RESOURCES: {
										name:  "SPIGET_RESOURCES"
										value: strings.Join(_spigetStrings, ",")
									}
								}
								if _c.paper.plugins.modrinth != _|_ {
									MODRINTH_PROJECTS: {
										name:  "MODRINTH_PROJECTS"
										value: strings.Join(_c.paper.plugins.modrinth.projects, ",")
									}
									if _c.paper.plugins.modrinth.downloadDependencies != _|_ {
										MODRINTH_DOWNLOAD_DEPENDENCIES: {
											name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
											value: _c.paper.plugins.modrinth.downloadDependencies
										}
									}
									if _c.paper.plugins.modrinth.allowedVersionType != _|_ {
										MODRINTH_ALLOWED_VERSION_TYPE: {
											name:  "MODRINTH_ALLOWED_VERSION_TYPE"
											value: _c.paper.plugins.modrinth.allowedVersionType
										}
									}
								}
								REMOVE_OLD_MODS: {
									name:  "REMOVE_OLD_MODS"
									value: "\(_c.paper.plugins.removeOldMods)"
								}
							}
						}
						if _c.spigot != _|_ {
							if _c.spigot.downloadUrl != _|_ {
								SPIGOT_DOWNLOAD_URL: {
									name:  "SPIGOT_DOWNLOAD_URL"
									value: _c.spigot.downloadUrl
								}
							}
							if _c.spigot.plugins != _|_ {
								if _c.spigot.plugins.urls != _|_ {
									PLUGINS: {
										name:  "PLUGINS"
										value: strings.Join(_c.spigot.plugins.urls, ",")
									}
								}
								if _c.spigot.plugins.spigetResources != _|_ {
									let _spigetStrings = [for r in _c.spigot.plugins.spigetResources {"\(r)"}]
									SPIGET_RESOURCES: {
										name:  "SPIGET_RESOURCES"
										value: strings.Join(_spigetStrings, ",")
									}
								}
								if _c.spigot.plugins.modrinth != _|_ {
									MODRINTH_PROJECTS: {
										name:  "MODRINTH_PROJECTS"
										value: strings.Join(_c.spigot.plugins.modrinth.projects, ",")
									}
									if _c.spigot.plugins.modrinth.downloadDependencies != _|_ {
										MODRINTH_DOWNLOAD_DEPENDENCIES: {
											name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
											value: _c.spigot.plugins.modrinth.downloadDependencies
										}
									}
									if _c.spigot.plugins.modrinth.allowedVersionType != _|_ {
										MODRINTH_ALLOWED_VERSION_TYPE: {
											name:  "MODRINTH_ALLOWED_VERSION_TYPE"
											value: _c.spigot.plugins.modrinth.allowedVersionType
										}
									}
								}
								REMOVE_OLD_MODS: {
									name:  "REMOVE_OLD_MODS"
									value: "\(_c.spigot.plugins.removeOldMods)"
								}
							}
						}
						if _c.bukkit != _|_ {
							if _c.bukkit.downloadUrl != _|_ {
								BUKKIT_DOWNLOAD_URL: {
									name:  "BUKKIT_DOWNLOAD_URL"
									value: _c.bukkit.downloadUrl
								}
							}
							if _c.bukkit.plugins != _|_ {
								if _c.bukkit.plugins.urls != _|_ {
									PLUGINS: {
										name:  "PLUGINS"
										value: strings.Join(_c.bukkit.plugins.urls, ",")
									}
								}
								if _c.bukkit.plugins.spigetResources != _|_ {
									let _spigetStrings = [for r in _c.bukkit.plugins.spigetResources {"\(r)"}]
									SPIGET_RESOURCES: {
										name:  "SPIGET_RESOURCES"
										value: strings.Join(_spigetStrings, ",")
									}
								}
								if _c.bukkit.plugins.modrinth != _|_ {
									MODRINTH_PROJECTS: {
										name:  "MODRINTH_PROJECTS"
										value: strings.Join(_c.bukkit.plugins.modrinth.projects, ",")
									}
									if _c.bukkit.plugins.modrinth.downloadDependencies != _|_ {
										MODRINTH_DOWNLOAD_DEPENDENCIES: {
											name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
											value: _c.bukkit.plugins.modrinth.downloadDependencies
										}
									}
									if _c.bukkit.plugins.modrinth.allowedVersionType != _|_ {
										MODRINTH_ALLOWED_VERSION_TYPE: {
											name:  "MODRINTH_ALLOWED_VERSION_TYPE"
											value: _c.bukkit.plugins.modrinth.allowedVersionType
										}
									}
								}
								REMOVE_OLD_MODS: {
									name:  "REMOVE_OLD_MODS"
									value: "\(_c.bukkit.plugins.removeOldMods)"
								}
							}
						}
						if _c.sponge != _|_ {
							SPONGEVERSION: {
								name:  "SPONGEVERSION"
								value: _c.sponge.version
							}
						}
						if _c.purpur != _|_ {
							if _c.purpur.plugins != _|_ {
								if _c.purpur.plugins.urls != _|_ {
									PLUGINS: {
										name:  "PLUGINS"
										value: strings.Join(_c.purpur.plugins.urls, ",")
									}
								}
								if _c.purpur.plugins.spigetResources != _|_ {
									let _spigetStrings = [for r in _c.purpur.plugins.spigetResources {"\(r)"}]
									SPIGET_RESOURCES: {
										name:  "SPIGET_RESOURCES"
										value: strings.Join(_spigetStrings, ",")
									}
								}
								if _c.purpur.plugins.modrinth != _|_ {
									MODRINTH_PROJECTS: {
										name:  "MODRINTH_PROJECTS"
										value: strings.Join(_c.purpur.plugins.modrinth.projects, ",")
									}
									if _c.purpur.plugins.modrinth.downloadDependencies != _|_ {
										MODRINTH_DOWNLOAD_DEPENDENCIES: {
											name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
											value: _c.purpur.plugins.modrinth.downloadDependencies
										}
									}
									if _c.purpur.plugins.modrinth.allowedVersionType != _|_ {
										MODRINTH_ALLOWED_VERSION_TYPE: {
											name:  "MODRINTH_ALLOWED_VERSION_TYPE"
											value: _c.purpur.plugins.modrinth.allowedVersionType
										}
									}
								}
								REMOVE_OLD_MODS: {
									name:  "REMOVE_OLD_MODS"
									value: "\(_c.purpur.plugins.removeOldMods)"
								}
							}
						}
						if _c.magma != _|_ {
							if _c.magma.mods != _|_ {
								if _c.magma.mods.urls != _|_ {
									MODS: {
										name:  "MODS"
										value: strings.Join(_c.magma.mods.urls, ",")
									}
								}
								if _c.magma.mods.modrinth != _|_ {
									MODRINTH_PROJECTS: {
										name:  "MODRINTH_PROJECTS"
										value: strings.Join(_c.magma.mods.modrinth.projects, ",")
									}
									if _c.magma.mods.modrinth.downloadDependencies != _|_ {
										MODRINTH_DOWNLOAD_DEPENDENCIES: {
											name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
											value: _c.magma.mods.modrinth.downloadDependencies
										}
									}
									if _c.magma.mods.modrinth.allowedVersionType != _|_ {
										MODRINTH_ALLOWED_VERSION_TYPE: {
											name:  "MODRINTH_ALLOWED_VERSION_TYPE"
											value: _c.magma.mods.modrinth.allowedVersionType
										}
									}
								}
								if _c.magma.mods.modpackUrl != _|_ {
									MODPACK: {
										name:  "MODPACK"
										value: _c.magma.mods.modpackUrl
									}
								}
								REMOVE_OLD_MODS: {
									name:  "REMOVE_OLD_MODS"
									value: "\(_c.magma.mods.removeOldMods)"
								}
							}
							if _c.magma.plugins != _|_ {
								if _c.magma.plugins.urls != _|_ {
									PLUGINS: {
										name:  "PLUGINS"
										value: strings.Join(_c.magma.plugins.urls, ",")
									}
								}
								if _c.magma.plugins.spigetResources != _|_ {
									let _spigetStrings = [for r in _c.magma.plugins.spigetResources {"\(r)"}]
									SPIGET_RESOURCES: {
										name:  "SPIGET_RESOURCES"
										value: strings.Join(_spigetStrings, ",")
									}
								}
							}
						}
						if _c.ftba != _|_ {
							if _c.ftba.mods != _|_ {
								if _c.ftba.mods.urls != _|_ {
									MODS: {
										name:  "MODS"
										value: strings.Join(_c.ftba.mods.urls, ",")
									}
								}
								if _c.ftba.mods.modrinth != _|_ {
									MODRINTH_PROJECTS: {
										name:  "MODRINTH_PROJECTS"
										value: strings.Join(_c.ftba.mods.modrinth.projects, ",")
									}
									if _c.ftba.mods.modrinth.downloadDependencies != _|_ {
										MODRINTH_DOWNLOAD_DEPENDENCIES: {
											name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
											value: _c.ftba.mods.modrinth.downloadDependencies
										}
									}
									if _c.ftba.mods.modrinth.allowedVersionType != _|_ {
										MODRINTH_ALLOWED_VERSION_TYPE: {
											name:  "MODRINTH_ALLOWED_VERSION_TYPE"
											value: _c.ftba.mods.modrinth.allowedVersionType
										}
									}
								}
								if _c.ftba.mods.modpackUrl != _|_ {
									MODPACK: {
										name:  "MODPACK"
										value: _c.ftba.mods.modpackUrl
									}
								}
								REMOVE_OLD_MODS: {
									name:  "REMOVE_OLD_MODS"
									value: "\(_c.ftba.mods.removeOldMods)"
								}
							}
						}
						if _c.autoCurseForge != _|_ {
							CF_API_KEY: {
								name: "CF_API_KEY"
								from: _c.autoCurseForge.apiKey
							}
							if _c.autoCurseForge.pageUrl != _|_ {
								CF_PAGE_URL: {
									name:  "CF_PAGE_URL"
									value: _c.autoCurseForge.pageUrl
								}
							}
							if _c.autoCurseForge.slug != _|_ {
								CF_SLUG: {
									name:  "CF_SLUG"
									value: _c.autoCurseForge.slug
								}
							}
							if _c.autoCurseForge.fileId != _|_ {
								CF_FILE_ID: {
									name:  "CF_FILE_ID"
									value: _c.autoCurseForge.fileId
								}
							}
							if _c.autoCurseForge.filenameMatcher != _|_ {
								CF_FILENAME_MATCHER: {
									name:  "CF_FILENAME_MATCHER"
									value: _c.autoCurseForge.filenameMatcher
								}
							}
							if _c.autoCurseForge.forceSynchronize != _|_ {
								CF_FORCE_SYNCHRONIZE: {
									name:  "CF_FORCE_SYNCHRONIZE"
									value: "\(_c.autoCurseForge.forceSynchronize)"
								}
							}
							if _c.autoCurseForge.parallelDownloads != _|_ {
								CF_PARALLEL_DOWNLOADS: {
									name:  "CF_PARALLEL_DOWNLOADS"
									value: "\(_c.autoCurseForge.parallelDownloads)"
								}
							}
						}

						// === Modrinth Modpack ===
						if _c.modrinth != _|_ {
							MODRINTH_MODPACK: {
								name:  "MODRINTH_MODPACK"
								value: _c.modrinth.modpack
							}
							if _c.modrinth.version != _|_ {
								MODRINTH_VERSION: {
									name:  "MODRINTH_VERSION"
									value: _c.modrinth.version
								}
							}
							if _c.modrinth.projects != _|_ {
								MODRINTH_PROJECTS: {
									name:  "MODRINTH_PROJECTS"
									value: strings.Join(_c.modrinth.projects, ",")
								}
							}
							if _c.modrinth.downloadDependencies != _|_ {
								MODRINTH_DOWNLOAD_DEPENDENCIES: {
									name:  "MODRINTH_DOWNLOAD_DEPENDENCIES"
									value: _c.modrinth.downloadDependencies
								}
							}
						}

						// === Resource Packs ===
						if _c.server.resourcePackUrl != _|_ {
							RESOURCE_PACK: {
								name:  "RESOURCE_PACK"
								value: _c.server.resourcePackUrl
							}
						}
						if _c.server.resourcePackSha != _|_ {
							RESOURCE_PACK_SHA1: {
								name:  "RESOURCE_PACK_SHA1"
								value: _c.server.resourcePackSha
							}
						}
						if _c.server.resourcePackEnforce != _|_ {
							RESOURCE_PACK_ENFORCE: {
								name:  "RESOURCE_PACK_ENFORCE"
								value: "\(_c.server.resourcePackEnforce)"
							}
						}

						// === VanillaTweaks ===
						if _c.server.vanillaTweaksShareCodes != _|_ {
							VANILLATWEAKS_SHARECODE: {
								name:  "VANILLATWEAKS_SHARECODE"
								value: strings.Join(_c.server.vanillaTweaksShareCodes, ",")
							}
						}

						// === World Data ===
						if _c.downloadWorldUrl != _|_ {
							WORLD: {
								name:  "WORLD"
								value: _c.downloadWorldUrl
							}
						}

						// === Query Port ===
						if _c.query.enabled {
							ENABLE_QUERY: {
								name:  "ENABLE_QUERY"
								value: "true"
							}
							QUERY_PORT: {
								name:  "QUERY_PORT"
								value: "\(_c.query.port)"
							}
						}
					}

					volumeMounts: {
						data: volumes.data & {
							mountPath: "/data"
						}
					}

					if _c.resources != _|_ {
						resources: _c.resources
					}

					// === Health Checks ===
					startupProbe: {
						exec: command: ["mc-monitor", "status", "--port", "\(_c.port)"]
						periodSeconds:    10
						timeoutSeconds:   5
						failureThreshold: 60
					}
					livenessProbe: {
						exec: command: ["mc-monitor", "status", "--port", "\(_c.port)"]
						periodSeconds:    30
						timeoutSeconds:   5
						failureThreshold: 3
					}
					readinessProbe: {
						exec: command: ["mc-monitor", "status", "--port", "\(_c.port)"]
						periodSeconds:    10
						timeoutSeconds:   3
						failureThreshold: 3
					}
				}

				// === Sidecar Containers ===
				let _backupSidecar = [if _c.backup.enabled {
					{
						name:  "backup"
						image: _c.backup.image

						env: {
							if _c.backup.tar != _|_ {
								BACKUP_METHOD: {name: "BACKUP_METHOD", value: "tar"}
							}
							if _c.backup.rsync != _|_ {
								BACKUP_METHOD: {name: "BACKUP_METHOD", value: "rsync"}
							}
							if _c.backup.restic != _|_ {
								BACKUP_METHOD: {name: "BACKUP_METHOD", value: "restic"}
							}
							if _c.backup.rclone != _|_ {
								BACKUP_METHOD: {name: "BACKUP_METHOD", value: "rclone"}
							}
							BACKUP_INTERVAL: {
								name:  "BACKUP_INTERVAL"
								value: _c.backup.interval
							}
							INITIAL_DELAY: {
								name:  "INITIAL_DELAY"
								value: _c.backup.initialDelay
							}
							RCON_HOST: {name: "RCON_HOST", value: "localhost"}
							RCON_PORT: {
								name:  "RCON_PORT"
								value: "\(_c.rcon.port)"
							}
							RCON_PASSWORD: {
								name: "RCON_PASSWORD"
								from: #config.rconPassword
							}
							SRC_DIR: {name: "SRC_DIR", value: "/data"}
							DEST_DIR: {name: "DEST_DIR", value: "/backups"}
							if _c.backup.pruneBackupsDays != _|_ {
								PRUNE_BACKUPS_DAYS: {
									name:  "PRUNE_BACKUPS_DAYS"
									value: "\(_c.backup.pruneBackupsDays)"
								}
							}
							PAUSE_IF_NO_PLAYERS: {
								name:  "PAUSE_IF_NO_PLAYERS"
								value: "\(_c.backup.pauseIfNoPlayers)"
							}
							if _c.backup.backupName != _|_ {
								BACKUP_NAME: {
									name:  "BACKUP_NAME"
									value: _c.backup.backupName
								}
							}
							if _c.backup.excludes != _|_ {
								EXCLUDES: {
									name:  "EXCLUDES"
									value: strings.Join(_c.backup.excludes, ",")
								}
							}
							if _c.backup.tar != _|_ {
								TAR_COMPRESS_METHOD: {
									name:  "TAR_COMPRESS_METHOD"
									value: _c.backup.tar.compressMethod
								}
								LINK_LATEST: {
									name:  "LINK_LATEST"
									value: "\(_c.backup.tar.linkLatest)"
								}
								if _c.backup.tar.compressParameters != _|_ {
									TAR_COMPRESS_PARAMETERS: {
										name:  "TAR_COMPRESS_PARAMETERS"
										value: _c.backup.tar.compressParameters
									}
								}
							}
							if _c.backup.rsync != _|_ {
								LINK_LATEST: {
									name:  "LINK_LATEST"
									value: "\(_c.backup.rsync.linkLatest)"
								}
							}
							if _c.backup.restic != _|_ {
								RESTIC_REPOSITORY: {
									name:  "RESTIC_REPOSITORY"
									value: _c.backup.restic.repository
								}
								RESTIC_PASSWORD: {
									name: "RESTIC_PASSWORD"
									from: _c.backup.restic.password
								}
								if _c.backup.restic.retention != _|_ {
									PRUNE_RESTIC_RETENTION: {
										name:  "PRUNE_RESTIC_RETENTION"
										value: _c.backup.restic.retention
									}
								}
								if _c.backup.restic.hostname != _|_ {
									RESTIC_HOSTNAME: {
										name:  "RESTIC_HOSTNAME"
										value: _c.backup.restic.hostname
									}
								}
								if _c.backup.restic.verbose != _|_ {
									RESTIC_VERBOSE: {
										name:  "RESTIC_VERBOSE"
										value: "\(_c.backup.restic.verbose)"
									}
								}
								if _c.backup.restic.additionalTags != _|_ {
									RESTIC_ADDITIONAL_TAGS: {
										name:  "RESTIC_ADDITIONAL_TAGS"
										value: _c.backup.restic.additionalTags
									}
								}
								if _c.backup.restic.limitUpload != _|_ {
									RESTIC_LIMIT_UPLOAD: {
										name:  "RESTIC_LIMIT_UPLOAD"
										value: "\(_c.backup.restic.limitUpload)"
									}
								}
								if _c.backup.restic.retryLock != _|_ {
									RESTIC_RETRY_LOCK: {
										name:  "RESTIC_RETRY_LOCK"
										value: _c.backup.restic.retryLock
									}
								}
								if _c.backup.restic.accessKey != _|_ {
									AWS_ACCESS_KEY_ID: {
										name: "AWS_ACCESS_KEY_ID"
										from: _c.backup.restic.accessKey
									}
								}
								if _c.backup.restic.secretKey != _|_ {
									AWS_SECRET_ACCESS_KEY: {
										name: "AWS_SECRET_ACCESS_KEY"
										from: _c.backup.restic.secretKey
									}
								}
							}
							if _c.backup.rclone != _|_ {
								RCLONE_REMOTE: {
									name:  "RCLONE_REMOTE"
									value: _c.backup.rclone.remote
								}
								RCLONE_DEST_DIR: {
									name:  "RCLONE_DEST_DIR"
									value: _c.backup.rclone.destDir
								}
								RCLONE_COMPRESS_METHOD: {
									name:  "RCLONE_COMPRESS_METHOD"
									value: _c.backup.rclone.compressMethod
								}
							}
						}

						volumeMounts: {
							data: volumes.data & {
								mountPath: "/data"
								readOnly:  true
							}
						if _c.backup.enabled && _c.backup.method == "tar" {
							backups: {
								name:      "backups"
								mountPath: "/backups"
								if _c.storage.backups.type == "pvc" {
									persistentClaim: {
										size: _c.storage.backups.size
										if _c.storage.backups.storageClass != _|_ {
											storageClass: _c.storage.backups.storageClass
										}
									}
								}
								if _c.storage.backups.type == "hostPath" {
									hostPath: {
										path: _c.storage.backups.path
										type: _c.storage.backups.hostPathType
									}
								}
								if _c.storage.backups.type == "emptyDir" {
									emptyDir: {}
								}
							}
						}
						if _c.backup.enabled && _c.backup.method == "rsync" {
							if _c.backup.rsync.useLocalStorage {
								backups: {
									name:      "backups"
									mountPath: "/backups"
									if _c.storage.backups.type == "pvc" {
										persistentClaim: {
											size: _c.storage.backups.size
											if _c.storage.backups.storageClass != _|_ {
												storageClass: _c.storage.backups.storageClass
											}
										}
									}
									if _c.storage.backups.type == "hostPath" {
										hostPath: {
											path: _c.storage.backups.path
											type: _c.storage.backups.hostPathType
										}
									}
									if _c.storage.backups.type == "emptyDir" {
										emptyDir: {}
									}
								}
							}
						}
						}

						resources: {
							requests: {cpu: "100m", memory: "256Mi"}
							limits: {cpu: "1000m", memory: "1Gi"}
						}
					}
				}]

				let _monitorSidecar = [if _c.monitor.enabled {
					{
						name:  "mc-monitor"
						image: _c.monitor.image
						command: ["/mc-monitor", "export-for-prometheus"]
						ports: metrics: {
							name:       "metrics"
							targetPort: _c.monitor.port
							protocol:   "TCP"
						}
						env: {
							EXPORT_SERVERS: {
								name: "EXPORT_SERVERS"
								// Use the K8s Service name for meaningful per-server labels.
								// Overridable via _c.monitor.serverHost; defaults to "localhost".
								value: "\(_c.monitor.serverHost):\(_c.port)"
							}
							EXPORT_PORT: {
								name:  "EXPORT_PORT"
								value: "\(_c.monitor.port)"
							}
							TIMEOUT: {
								name:  "TIMEOUT"
								value: _c.monitor.timeout
							}
						}
						resources: {
							requests: {cpu: "10m", memory: "32Mi"}
							limits: {cpu: "100m", memory: "64Mi"}
						}
					}
				}]

				sidecarContainers: list.Concat([_backupSidecar, _monitorSidecar])

				// === Network Exposure ===
				expose: {
					ports: {
						minecraft: {
							targetPort:  25565
							protocol:    "TCP"
							exposedPort: _c.port
						}
						if _c.monitor.enabled {
							metrics: {
								targetPort:  _c.monitor.port
								protocol:    "TCP"
								exposedPort: _c.monitor.port
							}
						}
					}
					type: _c.serviceType
				}

				// === Volumes ===
				volumes: {
					data: {
						name: "data"
						if _c.storage.data.type == "pvc" {
							persistentClaim: {
								size: _c.storage.data.size
								if _c.storage.data.storageClass != _|_ {
									storageClass: _c.storage.data.storageClass
								}
							}
						}
						if _c.storage.data.type == "hostPath" {
							hostPath: {
								path: _c.storage.data.path
								type: _c.storage.data.hostPathType
							}
						}
						if _c.storage.data.type == "emptyDir" {
							emptyDir: {}
						}
					}
					if _c.backup.enabled && _c.backup.method == "tar" {
						backups: {
							name: "backups"
							if _c.storage.backups.type == "pvc" {
								persistentClaim: {
									size: _c.storage.backups.size
									if _c.storage.backups.storageClass != _|_ {
										storageClass: _c.storage.backups.storageClass
									}
								}
							}
							if _c.storage.backups.type == "hostPath" {
								hostPath: {
									path: _c.storage.backups.path
									type: _c.storage.backups.hostPathType
								}
							}
							if _c.storage.backups.type == "emptyDir" {
								emptyDir: {}
							}
						}
					}
					if _c.backup.enabled && _c.backup.method == "rsync" {
						if _c.backup.rsync.useLocalStorage {
							backups: {
								name: "backups"
								if _c.storage.backups.type == "pvc" {
									persistentClaim: {
										size: _c.storage.backups.size
										if _c.storage.backups.storageClass != _|_ {
											storageClass: _c.storage.backups.storageClass
										}
									}
								}
								if _c.storage.backups.type == "hostPath" {
									hostPath: {
										path: _c.storage.backups.path
										type: _c.storage.backups.hostPathType
									}
								}
								if _c.storage.backups.type == "emptyDir" {
									emptyDir: {}
								}
							}
						}
					}
				}
			}
		}
	}

	// ── mc-router ─────────────────────────────────────────────────────────────
	// Always present. --mapping args auto-built from the servers map:
	//   {serverName}.{domain}  →  {releaseName}-server-{serverName}.{namespace}.svc:{port}
	router: {
		resources_workload.#Container
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_workload.#UpdateStrategy
		traits_network.#Expose
		traits_security.#WorkloadIdentity

		metadata: labels: "core.opmodel.dev/workload-type": "stateless"

		spec: {
			scaling: count: 1

			restartPolicy: "Always"

			updateStrategy: type: "Recreate"

			workloadIdentity: {
				name:           "mc-router"
				automountToken: true
			}

			container: {
				name:  "mc-router"
				image: #config.router.image

				ports: {
					minecraft: {
						targetPort: #config.router.port
						protocol:   "TCP"
					}
					if #config.router.api.enabled {
						api: {
							targetPort: #config.router.api.port
							protocol:   "TCP"
						}
					}
				}

				env: {
					PORT: {
						name:  "PORT"
						value: "\(#config.router.port)"
					}
					CONNECTION_RATE_LIMIT: {
						name:  "CONNECTION_RATE_LIMIT"
						value: "\(#config.router.connectionRateLimit)"
					}
					DEBUG: {
						name:  "DEBUG"
						value: "\(#config.router.debug)"
					}
					if #config.router.simplifySrv {
						SIMPLIFY_SRV: {
							name:  "SIMPLIFY_SRV"
							value: "true"
						}
					}
					if #config.router.useProxyProtocol {
						USE_PROXY_PROTOCOL: {
							name:  "USE_PROXY_PROTOCOL"
							value: "true"
						}
					}
					if #config.router.defaultServer != _|_ {
						DEFAULT: {
							name:  "DEFAULT"
							value: "\(#config.router.defaultServer.host):\(#config.router.defaultServer.port)"
						}
					}
					if #config.router.autoScale != _|_ {
						if #config.router.autoScale.up != _|_ {
							AUTO_SCALE_UP: {
								name:  "AUTO_SCALE_UP"
								value: "\(#config.router.autoScale.up.enabled)"
							}
						}
						if #config.router.autoScale.down != _|_ {
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
					}
					if #config.router.metrics != _|_ {
						METRICS_BACKEND: {
							name:  "METRICS_BACKEND"
							value: #config.router.metrics.backend
						}
					}
					if #config.router.api.enabled {
						API_BINDING: {
							name:  "API_BINDING"
							value: ":\(#config.router.api.port)"
						}
					}
				}

				// Auto-build --mapping args from the servers map.
				// Format: -mapping={serverName}.{domain}={releaseName}-server-{serverName}.{namespace}.svc:{port}
				args: [for _srvName, _srvCfg in #config.servers {
					"-mapping=\(_srvName).\(_domain)=\(_relName)-server-\(_srvName).\(_ns).svc:\(_srvCfg.port)"
				}]

				if #config.router.resources != _|_ {
					resources: #config.router.resources
				}
			}

			expose: {
				ports: {
					minecraft: {
						targetPort:  #config.router.port
						protocol:    "TCP"
						exposedPort: #config.router.port
					}
					if #config.router.api.enabled {
						api: {
							targetPort:  #config.router.api.port
							protocol:    "TCP"
							exposedPort: #config.router.api.port
						}
					}
				}
				type: #config.router.serviceType
			}
		}
	}

	// ── RBAC ──────────────────────────────────────────────────────────────────
	// Grants mc-router permission to watch/list Services (service discovery)
	// and manage StatefulSets (auto-scale wake/sleep).
	rbac: {
		resources_security.#Role

		spec: role: {
			name:  "mc-router"
			scope: "cluster"

			rules: [
				{
					apiGroups: [""]
					resources: ["services"]
					verbs: ["watch", "list"]
				},
				{
					apiGroups: ["apps"]
					resources: ["statefulsets", "statefulsets/scale"]
					verbs: ["watch", "list", "get", "update", "patch"]
				},
			]

			subjects: [{
				name: "mc-router"
			}]
		}
	}
}

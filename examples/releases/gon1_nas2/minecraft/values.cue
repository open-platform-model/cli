// Values for the minecraft-create fleet release on gon1_nas2.
// Converted from docker-compose (minecraft-create stack).
//
// Servers:
//   creative  →  creative.mc.larnet.eu  (Create Creative, Paper/Modrinth, 5G, no backup)
//   survival  →  survival.mc.larnet.eu  (Create Survival, Modrinth, 10G, restic backup)
//
// Both servers share the same Modrinth modpack (jonklscreatemodpack) at different versions.
// mc-router (LoadBalancer) routes player connections by hostname; defaults to creative.
package minecraft

values: {
	// Must match metadata.name above — used to compute K8s Service DNS names.
	releaseName: "mc"
	domain:      "mc.larnet.eu"
	namespace:   "minecraft"

	// === Shared RCON password ===
	// OPM will create a K8s Secret named "minecraft-create-server-secrets"
	// with key "rcon-password" containing this value.
	rconPassword: value: "changeme"

	// ── Servers ───────────────────────────────────────────────────────────────

	servers: {

		// ── create-survival ──────────────────────────────────────────────────────────
		// Routes: create-survival.mc.larnet.eu → minecraft-create-server-create-survival.minecraft.svc
		// Source: create-survival-server in docker-compose
		"create-survival": {
			enabled: true
			N=name: "create-survival"
			image: {
				repository: "itzg/minecraft-server"
				tag:        "java21"
				digest:     ""
			}

			version: "1.21.1"

			modrinth: {
				modpack:              "https://modrinth.com/modpack/create-ultimate-selection-2"
				version:              "Mun9yNz5"
				downloadDependencies: "required"
				// bluemap with pinned version
				// projects: ["bluemap:lHRktt6S"]
			}

			jvm: {
				initMemory:    "2G"
				maxMemory:     "5G"
				useAikarFlags: true
			}

			server: {
				motd:              "NorthByte Create Server - Survival"
				serverName:        "NorthByte Create - Survival"
				mode:              "survival"
				maxPlayers:        20
				difficulty:        "normal"
				pvp:               true
				allowFlight:       false
				enableRollingLogs: true
				seed:              "-1106759604738884840"
				spawnProtection:   15
				levelType:         "minecraft:normal"
				worldSaveName:     "world"
				allowNether:       true
				onlineMode:        true
				ops: [
					"032bb8dd-c4e6-411e-bce7-54379e9819c5", // Emil
				]
				tz: "Europe/Stockholm"
			}

			rcon: {
				enabled: true
				port:    25575
			}

			port:        25565
			serviceType: "ClusterIP"

			// BlueMap web map (served by the bluemap plugin inside the server container)
			// extraPorts: [{
			// 	name:          "bluemap"
			// 	containerPort: 8100
			// 	protocol:      "TCP"
			// }]

			storage: data: {
				type:         "hostPath"
				// This path is on the 240GB NVMe drive. TODO: migrate to 2TB ZFS later
				path:         "/var/local-path-provisioner/minecraft/prod/create-survival"
				hostPathType: "DirectoryOrCreate"
			}

			backup: {
				enabled:          true
				method:           "restic"
				interval:         "1h"
				initialDelay:     "2m"
				pruneBackupsDays: 20
				pauseIfNoPlayers: true
				backupName:       "\(releaseName)-\(N)-backup"
				excludes: ["./bluemap/*", "./plugins/CoreProtect/*"]

				restic: {
					repository: "s3:http://10.10.0.2:30304/mc-backup/create-survival"
					password: value:  "9FphluY#^0XiEhaVb7H4urkaj0ZPS8"
					accessKey: value: "ABIA0A4Y35DTP50LWJHV"
					secretKey: value: "kxR3l1hNww1C2nLaPjIqZZMeYErKgi0RPpTSHXCz"
					retention: "--keep-within 20d"
				}
			}

			monitor: enabled: true
		}

		// ── create-creative ──────────────────────────────────────────────────────────
		// Routes: create-creative.mc.larnet.eu → minecraft-create-server-create-creative.minecraft.svc
		// Source: create-creative-server in docker-compose
		"create-creative": {
			enabled: true
			N=name: "create-creative"
			image: {
				repository: "itzg/minecraft-server"
				tag:        "java21"
				digest:     ""
			}

			version: "1.21.1"

			modrinth: {
				modpack:              "https://modrinth.com/modpack/create-ultimate-selection-2"
				version:              "Mun9yNz5"
				downloadDependencies: "required"
				// bluemap with pinned version
				// projects: ["bluemap:lHRktt6S"]
			}

			jvm: {
				initMemory:    "2G"
				maxMemory:     "5G"
				useAikarFlags: true
			}

			server: {
				motd:              "NorthByte Create Server - Creative"
				serverName:        "NorthByte Create - Creative"
				mode:              "creative"
				maxPlayers:        20
				difficulty:        "normal"
				pvp:               true
				allowFlight:       false
				enableRollingLogs: true
				seed:              "-5831362641909587104"
				spawnProtection:   15
				levelType:         "minecraft:normal"
				worldSaveName:     "world"
				allowNether:       true
				onlineMode:        true
				ops: [
					"032bb8dd-c4e6-411e-bce7-54379e9819c5", // Emil
				]
				tz: "Europe/Stockholm"
			}

			rcon: {
				enabled: true
				port:    25575
			}

			port:        25565
			serviceType: "ClusterIP"

			// BlueMap web map (served by the bluemap plugin inside the server container)
			// extraPorts: [{
			// 	name:          "bluemap"
			// 	containerPort: 8100
			// 	protocol:      "TCP"
			// }]

			storage: data: {
				type:         "hostPath"
				// This path is on the 240GB NVMe drive. TODO: migrate to 2TB ZFS later
				path:         "/var/local-path-provisioner/minecraft/prod/create-creative"
				hostPathType: "DirectoryOrCreate"
			}

			backup: {
				enabled:          true
				method:           "restic"
				interval:         "1h"
				initialDelay:     "2m"
				pruneBackupsDays: 20
				pauseIfNoPlayers: true
				backupName:       "\(releaseName)-\(N)-backup"
				excludes: ["./bluemap/*", "./plugins/CoreProtect/*"]

				restic: {
					repository: "s3:http://10.10.0.2:30304/mc-backup/create-creative"
					password: value:  "9FphluY#^0XiEhaVb7H4urkaj0ZPS8"
					accessKey: value: "ABIA0A4Y35DTP50LWJHV"
					secretKey: value: "kxR3l1hNww1C2nLaPjIqZZMeYErKgi0RPpTSHXCz"
					retention: "--keep-within 20d"
				}
			}

			monitor: enabled: true
		}

		// ── vanilla-prod ──────────────────────────────────────────────────────────
		// Routes: vanilla-prod.mc-dev.larnet.eu → mc-dev-server-vanilla-prod.minecraft.svc
		// Source: vanilla-prod-server in docker-compose
		"vanilla": {
			enabled: true
			N=name:  "vanilla"
			image: {
				repository: "itzg/minecraft-server"
				tag:        "java21"
				digest:     ""
			}

			version: "1.21.11"

			paper: {
				plugins: {
					modrinth: {
						// bluemap:Vb2ZE8bR                    = 5.16        https://modrinth.com/plugin/bluemap/version/5.16-paper
						// essentialsx:Oa9ZDzZq                = 2.21.2      https://modrinth.com/plugin/essentialsx/version/2.21.2
						// luckperms:OrIs0S6b                  = v5.5.17     https://modrinth.com/plugin/luckperms/version/v5.5.17-bukkit
						// multiverse-core:68aSO5t5            = 5.5.3       https://modrinth.com/plugin/multiverse-core/version/5.5.3
						// multiverse-inventories:lvgetpFU     = 5.3.2       https://modrinth.com/plugin/multiverse-inventories/version/5.3.2
						// multiverse-portals:Qy5wD65R         = 5.2.1       https://modrinth.com/plugin/multiverse-portals/version/5.2.1
						// multiverse-netherportals:JVd0QB5k   = 5.0.4       https://modrinth.com/plugin/multiverse-netherportals/version/5.0.4
						// essentialsx-chat-module:BdLUtz0O    = 2.21.2      https://modrinth.com/plugin/essentialsx-chat-module/version/2.21.2
						// essentialsx-spawn:RVbLg2Am          = 2.21.2      https://modrinth.com/plugin/essentialsx-spawn/version/2.21.2
						projects: [
							"bluemap:Vb2ZE8bR",
							"essentialsx:Oa9ZDzZq",
							"essentialsx-chat-module:BdLUtz0O",
							"essentialsx-spawn:RVbLg2Am",
							"luckperms:OrIs0S6b",
							"multiverse-core:68aSO5t5",
							"multiverse-inventories:lvgetpFU",
							"multiverse-portals:Qy5wD65R",
							"multiverse-netherportals:JVd0QB5k",
						]
						downloadDependencies: "required"
					}
					removeOldMods: true
				}
			}

			jvm: {
				initMemory:    "2G"
				maxMemory:     "4G"
				useAikarFlags: true
			}

			server: {
				motd:              "NorthByte Vanilla Server"
				serverName:        "NorthByte Vanilla"
				mode:              "survival"
				maxPlayers:        20
				difficulty:        "normal"
				pvp:               true
				allowFlight:       false
				enableRollingLogs: true
				seed:              "-5831362641909587104"
				spawnProtection:   15
				levelType:         "minecraft:normal"
				worldSaveName:     "world"
				allowNether:       true
				onlineMode:        true
				ops: [
					"032bb8dd-c4e6-411e-bce7-54379e9819c5", // Emil
				]
				tz: "Europe/Stockholm"
			}

			rcon: {
				enabled: true
				port:    25575
			}

			port:        25565
			serviceType: "ClusterIP"

			// BlueMap web map (served by the bluemap plugin inside the server container)
			extraPorts: [{
				name:          "bluemap"
				containerPort: 8100
				protocol:      "TCP"
				expose:        true
			}]

			// Additional hostnames the router maps to this server.
			// Each alias produces an extra --mapping arg pointing to the same
			// backend as the primary {serverName}.{domain} mapping.
			// Example: ["vanilla.larnet.eu", "mc.larnet.eu"]
			aliases: ["vanilla.larnet.eu", "mc1.larnet.eu"]

			storage: data: {
				type:         "hostPath"
				// This path is on the 240GB NVMe drive. TODO: migrate to 2TB ZFS later
				path:         "/var/local-path-provisioner/minecraft/prod/vanilla"
				hostPathType: "DirectoryOrCreate"
			}

			backup: {
				enabled:          true
				method:           "restic"
				interval:         "1h"
				initialDelay:     "2m"
				pruneBackupsDays: 20
				pauseIfNoPlayers: true
				backupName:       "\(releaseName)-\(N)-backup"
				excludes: ["./bluemap/*", "./plugins/CoreProtect/*"]

				restic: {
					repository: "s3:http://10.10.0.2:30304/mc-backup/vanilla"
					password: value:  "9FphluY#^0XiEhaVb7H4urkaj0ZPS8"
					accessKey: value: "ABIA0A4Y35DTP50LWJHV"
					secretKey: value: "kxR3l1hNww1C2nLaPjIqZZMeYErKgi0RPpTSHXCz"
					retention: "--keep-within 20d"
				}
			}

			monitor: enabled: true
		}
	}

	// ── Restic GUI (Backrest) ─────────────────────────────────────────────────
	// Backrest web UI pre-configured with both restic repos.
	// First deploy writes /data/config.json via the init container.
	// Access at http://<node-ip>:9898 — both create-creative and create-survival
	// repos are pre-loaded; click "Index Snapshots" in the UI to populate them.
	resticGui: {
		enabled:     true
		port:        9898
		serviceType: "ClusterIP"
		username:    "admin"
		password: value: "sustained-spendable-wrongly-capably-poise-task"
		storage: data: {
			type:         "pvc"
			size:         "5Gi"
			storageClass: "local-path"
		}
	}

	// ── Code Server ───────────────────────────────────────────────────────────
	// Single VS Code-in-browser instance mounting all server data volumes.
	// /servers/create-creative → /var/local-path-provisioner/minecraft/create-creative
	// /servers/create-survival → /var/local-path-provisioner/minecraft/create-survival
	// Home PVC persists extensions and settings across restarts.
	codeServer: {
		enabled:     true
		port:        8080
		serviceType: "ClusterIP"
		password: value: "selection-choice-jolliness-quill-elephant-exhale"
		storage: home: {
			type:         "pvc"
			size:         "30Gi"
			storageClass: "local-path"
		}
	}

	// ── Router ────────────────────────────────────────────────────────────────
	// Single LoadBalancer Service on port 25565 routing by hostname.
	// creative.mc.larnet.eu  →  minecraft-create-server-creative.minecraft-create.svc:25565
	// survival.mc.larnet.eu  →  minecraft-create-server-survival.minecraft-create.svc:25565
	router: {
		port:        25565
		serviceType: "LoadBalancer"
		// Players who connect without a matching hostname land on the survival server.
		defaultServer: {
			host: "\(releaseName)-server-vanilla.\(namespace).svc"
			port: 25565
		}
	}
}

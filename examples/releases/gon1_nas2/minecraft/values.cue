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
			N=name:  "create-survival"
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
				type: "hostPath"
				// This path is on the 240GB NVMe drive. TODO: migrate to 2TB ZFS later
				path:         "/var/local-path-provisioner/minecraft/prod/create-survival"
				hostPathType: "DirectoryOrCreate"
			}

			backup: {
				enabled:          true
				method:           "restic"
				interval:         "1h"
				initialDelay:     "10m"
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

			resources: {
				requests: {cpu: "100m", memory: "5Gi"}
				limits: {cpu: 2, memory: "5632Mi"}
			}
		}

		// ── create-creative ──────────────────────────────────────────────────────────
		// Routes: create-creative.mc.larnet.eu → minecraft-create-server-create-creative.minecraft.svc
		// Source: create-creative-server in docker-compose
		"create-creative": {
			enabled: true
			N=name:  "create-creative"
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
				type: "hostPath"
				// This path is on the 240GB NVMe drive. TODO: migrate to 2TB ZFS later
				path:         "/var/local-path-provisioner/minecraft/prod/create-creative"
				hostPathType: "DirectoryOrCreate"
			}

			backup: {
				enabled:          true
				method:           "restic"
				interval:         "1h"
				initialDelay:     "10m"
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

			resources: {
				requests: {cpu: "100m", memory: "5Gi"}
				limits: {cpu: 2, memory: "5632Mi"}
			}
		}

		// ── vanilla ──────────────────────────────────────────────────────────
		// Routes: vanilla.mc.larnet.eu → mc-server-vanilla.minecraft.svc
		// Source: vanilla-server in docker-compose
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
				type: "hostPath"
				// This path is on the 240GB NVMe drive. TODO: migrate to 2TB ZFS later
				path:         "/var/local-path-provisioner/minecraft/prod/vanilla"
				hostPathType: "DirectoryOrCreate"
			}

			backup: {
				enabled:          true
				method:           "restic"
				interval:         "1h"
				initialDelay:     "10m"
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

			resources: {
				requests: {cpu: "100m", memory: "4Gi"}
				limits: {cpu: 2, memory: "4608Mi"}
			}
		}

		// ── cobblemon ──────────────────────────────────────────────────────────
		// Routes: cobblemon.mc.larnet.eu → mc-server-cobblemon.minecraft.svc
		//         map.cobblemon.larnet.eu → bluemap (port 8100)
		// Source: cobblemon-server in docker-compose (cobblemon-fabric modpack)
		"cobblemon": {
			enabled: true
			N=name:  "cobblemon"
			image: {
				repository: "itzg/minecraft-server"
				tag:        "java21"
				digest:     ""
			}

			version: "1.21.1"

			modrinth: {
				modpack:              "https://modrinth.com/modpack/cobblemon-fabric"
				version:              "Lydu1ZNo"
				downloadDependencies: "required"
				// bluemap:Dr2hvJBc = pinned bluemap version for cobblemon-fabric
				projects: ["bluemap:Dr2hvJBc"]
			}

			jvm: {
				initMemory:    "2G"
				maxMemory:     "4G"
				useAikarFlags: true
			}

			server: {
				motd:              "NorthByte Cobblemon Server"
				serverName:        "NorthByte Cobblemon"
				mode:              "survival"
				maxPlayers:        20
				difficulty:        "normal"
				pvp:               true
				allowFlight:       false
				enableRollingLogs: true
				seed:              "2055796538"
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

			// BlueMap web map (served by the bluemap mod inside the server container)
			// Exposed via traefik at map.cobblemon.larnet.eu
			extraPorts: [{
				name:          "bluemap"
				containerPort: 8100
				protocol:      "TCP"
				expose:        true
			}]

			aliases: ["cobblemon.larnet.eu"]

			storage: data: {
				type:         "hostPath"
				path:         "/var/local-path-provisioner/minecraft/prod/cobblemon"
				hostPathType: "DirectoryOrCreate"
			}

			backup: {
				enabled:          true
				method:           "restic"
				interval:         "1h"
				initialDelay:     "10m"
				pruneBackupsDays: 20
				pauseIfNoPlayers: true
				backupName:       "\(releaseName)-\(N)-backup"
				excludes: ["./bluemap/*", "./plugins/CoreProtect/*"]

				restic: {
					repository: "s3:http://10.10.0.2:30304/mc-backup/cobblemon"
					password: value:  "9FphluY#^0XiEhaVb7H4urkaj0ZPS8"
					accessKey: value: "ABIA0A4Y35DTP50LWJHV"
					secretKey: value: "kxR3l1hNww1C2nLaPjIqZZMeYErKgi0RPpTSHXCz"
					retention: "--keep-within 20d"
				}
			}

			monitor: enabled: true

			resources: {
				requests: {cpu: "100m", memory: "2Gi"}
				limits: {cpu: 2, memory: "4608Mi"}
			}
		}

		// ── ron ──────────────────────────────────────────────────────────
		// Routes: ron.mc.larnet.eu → mc-server-ron.minecraft.svc
		//         map.ron.larnet.eu → bluemap (port 8100)
		// Source: ron-server in docker-compose (ron-fabric modpack)
		"ron": {
			enabled: true
			name:  "ron"
			image: {
				repository: "itzg/minecraft-server"
				tag:        "java21"
				digest:     ""
			}

			version: "1.20.1"

			modrinth: {
				modpack:              "https://modrinth.com/modpack/reign-of-nether-optimized"
				version:              "r5qzFdYg"
				downloadDependencies: "required"
			}

			jvm: {
				initMemory:    "2G"
				maxMemory:     "4G"
				useAikarFlags: true
			}

			server: {
				motd:              "NorthByte Reign of Nether Server"
				serverName:        "NorthByte Reign of Nether"
				mode:              "creative"
				maxPlayers:        20
				difficulty:        "normal"
				pvp:               true
				allowFlight:       false
				enableRollingLogs: true
				seed:              "a45n3546nas456na3456"
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

			// storage: data: {
			// 	type:         "hostPath"
			// 	path:         "/var/local-path-provisioner/minecraft/prod/ron"
			// 	hostPathType: "DirectoryOrCreate"
			// }
			storage: data: {
				type:         "emptyDir"
			}

			backup: {
				enabled:          false
			}

			monitor: enabled: true

			resources: {
				requests: {cpu: "100m", memory: "2Gi"}
				limits: {cpu: 2, memory: "4608Mi"}
			}
		}
	}

	// ── Restic GUI (Backrest) ─────────────────────────────────────────────────
	// Backrest web UI pre-configured with all restic repos (one per server with
	// backup.restic configured). Config is generated in CUE and stored as an
	// immutable K8s Secret — adding or removing servers automatically updates it.
	// Access at http://<node-ip>:9898 and click "Index Snapshots" to populate.
	resticGui: {
		enabled:            true
		port:               9898
		serviceType:        "ClusterIP"
		username:           "admin"
		passwordBcryptHash: "JDJhJDEwJENWRm9Nd1JSUmRqQ2NNR3NiTlJ1aGV3eGRKTDVrNTVBMUVDbzFjaHBZdlBCYjQyWFF1dzJt"
		multihostIdentity: {
			keyId:   "ecdsa.agbRJ8c5cciPFCT_3Yys1aRZQ_In9tb4bIO9lQV57Gs"
			privKey: "-----BEGIN EC PRIVATE-----\nMHcCAQEEIHK7bYTV+1aKsjJ9Ni1fB8HanLoOCm4feEK77k8gmu2HoAoGCCqGSM49\nAwEHoUQDQgAE58W0BRx5LqfIQpXAK9NopSzbGlN+CUeTFGSRVRBjxLiZLPpuAutc\n2HJZxToxBrHDlHqTNF6z7jC6odsnD7Bl5g==\n-----END EC PRIVATE-----\n"
			pubKey:  "-----BEGIN EC PUBLIC-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE58W0BRx5LqfIQpXAK9NopSzbGlN+\nCUeTFGSRVRBjxLiZLPpuAutc2HJZxToxBrHDlHqTNF6z7jC6odsnD7Bl5g==\n-----END EC PUBLIC-----\n"
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

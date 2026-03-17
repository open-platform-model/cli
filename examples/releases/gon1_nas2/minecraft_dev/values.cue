// Values for the minecraft development fleet release on gon1_nas2.
// Servers:
//   dev  →  dev.mc.larnet.eu  (Create Dev, Paper/Modrinth, 5G, no backup)
//
package minecraft_dev

values: {
	// Must match metadata.name above — used to compute K8s Service DNS names.
	releaseName: "mc-dev"
	domain:      "mc-dev.larnet.eu"
	namespace:   "default"

	// === Shared RCON password ===
	// OPM will create a K8s Secret named "minecraft-create-server-secrets"
	// with key "rcon-password" containing this value.
	rconPassword: value: "changeme"

	// ── Servers ───────────────────────────────────────────────────────────────

	servers: {

		// ── vanilla ──────────────────────────────────────────────────────────
		// Routes: vanilla.mc-dev.larnet.eu → mc-dev-server-vanilla.minecraft.svc
		// Source: vanilla-server in docker-compose
		"vanilla": {
			enabled: true
			name:    "vanilla"
			image: {
				repository: "itzg/minecraft-server"
				tag:        "java21"
				digest:     ""
			}

			version: "1.21.11"

			paper: {
				plugins: {
					modrinth: {
						// bluemap:Vb2ZE8bR                    = 5.16-paper  https://modrinth.com/plugin/bluemap/version/5.16-paper
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
				motd:              "NorthByte Vanilla Server - Dev"
				serverName:        "NorthByte Vanilla - Dev"
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

			storage: data: {
				type:         "hostPath"
				path:         "/var/local-path-provisioner/minecraft/vanilla-dev"
				hostPathType: "DirectoryOrCreate"
			}
			// storage: data: {
			// 	type:         "pvc"
			// 	size:         "5Gi"
			// 	storageClass: "local-path"
			// }

			// Bootstrap from the vanilla-survival world archive so the dev server
			// starts with a copy of the survival world. Worlds are skipped if they
			// already exist in /data — set force: true for a deliberate reset.
			bootstrap: {
				url: "http://10.10.0.2:30303/buckets/mc-worlds/vanilla-papermc/vanilla-bootstrap.tar.gz"
				// force: true  // uncomment to overwrite existing worlds
			}

			backup: {
				enabled: false
			}

			monitor: enabled: true
		}

	}

	// ── Restic GUI (Backrest) ─────────────────────────────────────────────────
	// Backrest web UI pre-configured with both restic repos.
	// First deploy writes /data/config.json via the init container.
	// repos are pre-loaded; click "Index Snapshots" in the UI to populate them.
	resticGui: {
		enabled:     false
		port:        9898
		serviceType: "ClusterIP"
		username:    "admin"
		password: value: "changeme"
		storage: data: {
			type:         "pvc"
			size:         "5Gi"
			storageClass: "local-path"
		}
	}

	// ── Code Server ───────────────────────────────────────────────────────────
	// Single VS Code-in-browser instance mounting all server data volumes.
	// /servers/vanilla → /var/local-path-provisioner/minecraft/vanilla-dev
	// /servers/create → /var/local-path-provisioner/minecraft/create-dev
	// Home PVC persists extensions and settings across restarts.
	codeServer: {
		enabled:     true
		port:        8080
		serviceType: "ClusterIP"
		password: value: "changeme"
		storage: home: {
			type:         "pvc"
			size:         "10Gi"
			storageClass: "local-path"
		}
	}

	// ── Router ────────────────────────────────────────────────────────────────
	// Single LoadBalancer Service on port 25565 routing by hostname.
	// vanilla.mc-dev.larnet.eu  →  mc-dev-server-vanilla.minecraft.svc:25565
	router: {
		port:        25565
		serviceType: "LoadBalancer"
		debug:       true // add extra labels for debugging
		// autoScaler: {
		// 	up: enabled: true
		// 	down: {
		// 		enabled: true
		// 		after?:  "10m"
		// 	}
		// }
		// Players who connect without a matching hostname land on the dev server.
		defaultServer: {
			host: "\(releaseName)-server-vanilla-prod.\(namespace).svc"
			port: 25565
		}
	}
}

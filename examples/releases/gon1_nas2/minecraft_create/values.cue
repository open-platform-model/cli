// Values for the minecraft-create fleet release on gon1_nas2.
// Converted from docker-compose (minecraft-create stack).
//
// Servers:
//   creative  →  creative.mc.larnet.eu  (Create Creative, Paper/Modrinth, 5G, no backup)
//   survival  →  survival.mc.larnet.eu  (Create Survival, Modrinth, 10G, restic backup)
//
// Both servers share the same Modrinth modpack (jonklscreatemodpack) at different versions.
// mc-router (LoadBalancer) routes player connections by hostname; defaults to creative.
package minecraft_create

values: {
	// Must match metadata.name above — used to compute K8s Service DNS names.
	releaseName: "minecraft-create"
	domain:      "mc.larnet.eu"
	namespace:   "minecraft-create"

	// === Shared RCON password ===
	// OPM will create a K8s Secret named "minecraft-create-server-secrets"
	// with key "rcon-password" containing this value.
	rconPassword: value: "changeme"

	// ── Servers ───────────────────────────────────────────────────────────────

	servers: {

		// ── creative ──────────────────────────────────────────────────────────
		// Routes: creative.mc.larnet.eu → minecraft-create-server-creative.minecraft-create.svc
		// Source: create-creative-server in docker-compose
		creative: {
			image: {
				repository: "itzg/minecraft-server"
				tag:        "java17"
				digest:     ""
			}

			version: "1.20.1"

			modrinth: {
				modpack:              "https://modrinth.com/modpack/jonklscreatemodpack"
				version:              "ZYF5kPnk"
				downloadDependencies: "required"
				// bluemap with pinned version
				projects: ["bluemap:lHRktt6S"]
			}

			jvm: {
				initMemory:    "2G"
				maxMemory:     "5G"
				useAikarFlags: true
			}

			server: {
				motd:              "A Create Creative Server"
				serverName:        "Larnet Create Creative"
				mode:              "creative"
				maxPlayers:        20
				difficulty:        "normal"
				pvp:               true
				allowFlight:       false
				enableRollingLogs: true
				seed:              "arvidleialego"
				spawnProtection:   15
				levelType:         "minecraft:normal"
				worldSaveName:     "world"
				allowNether:       true
				onlineMode:        true
				ops: ["032bb8dd-c4e6-411e-bce7-54379e9819c5"]
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
			}]

			storage: data: {
				type: "hostPath"
				path: "/mnt/nvme/apps/minecraft-servers/create-creative"
			}

			backup: enabled: false
		}

		// ── survival ──────────────────────────────────────────────────────────
		// Routes: survival.mc.larnet.eu → minecraft-create-server-survival.minecraft-create.svc
		// Source: create-surv-server + create-surv-backup in docker-compose
		survival: {
			image: {
				repository: "itzg/minecraft-server"
				tag:        "java17"
				digest:     ""
			}

			version: "1.20.1"

			modrinth: {
				modpack:              "https://modrinth.com/modpack/jonklscreatemodpack"
				version:              "rlTgMYB4"
				downloadDependencies: "required"
			}

			jvm: {
				// Split heap: INIT_MEMORY=2G, MEMORY=10G (maxMemory is rendered as MEMORY)
				initMemory: "2G"
				maxMemory:  "5G"
				// xxOpts:        "-XX:+UseG1GC -XX:+ParallelRefProcEnabled -XX:MaxGCPauseMillis=200 -XX:+UnlockExperimentalVMOptions -XX:+DisableExplicitGC -XX:G1NewSizePercent=30 -XX:G1MaxNewSizePercent=40 -XX:G1HeapRegionSize=8M -XX:G1ReservePercent=20 -XX:G1HeapWastePercent=5 -XX:G1MixedGCCountTarget=4 -XX:InitiatingHeapOccupancyPercent=15 -XX:G1MixedGCLiveThresholdPercent=90 -XX:G1RSetUpdatingPauseTimePercent=5 -XX:SurvivorRatio=32 -XX:+PerfDisableSharedMem -XX:MaxTenuringThreshold=1"
				useAikarFlags: true
			}

			server: {
				motd:              "A Create Survival Server"
				serverName:        "Larnet Create Survival"
				mode:              "survival"
				maxPlayers:        20
				difficulty:        "normal"
				pvp:               true
				allowFlight:       false
				enableRollingLogs: true
				seed:              "arvidleialego"
				spawnProtection:   15
				levelType:         "minecraft:normal"
				worldSaveName:     "world"
				allowNether:       true
				onlineMode:        true
				ops: ["032bb8dd-c4e6-411e-bce7-54379e9819c5"]
				tz: "Europe/Stockholm"
			}

			rcon: {
				enabled: true
				port:    25575
			}

			port:        25565
			serviceType: "ClusterIP"

			// BlueMap web map + Plan stats dashboard (served by plugins inside the container)
			extraPorts: [
				{
					name:          "bluemap"
					containerPort: 8100
					protocol:      "TCP"
				},
				{
					name:          "plan"
					containerPort: 8804
					protocol:      "TCP"
				},
			]

			storage: {
				data: {
					type: "hostPath"
					path: "/mnt/nvme/apps/minecraft-servers/create-surv"
				}
				backups: {
					type: "hostPath"
					path: "/mnt/nvme/apps/minecraft-servers/backups/create-surv"
				}
			}

			backup: {
				enabled:          true
				method:           "restic"
				interval:         "24h"
				initialDelay:     "2m"
				pruneBackupsDays: 20
				pauseIfNoPlayers: false
				backupName:       "world"
				excludes: ["./bluemap/*", "./plugins/CoreProtect/*"]

				restic: {
					// TODO: set your restic repository URL (e.g. "s3:s3.amazonaws.com/my-bucket")
					repository: "changeme"
					password: value: "changeme"
					retention: "--keep-within 20d"
				}
			}
		}
	}

	// ── Router ────────────────────────────────────────────────────────────────
	// Single LoadBalancer Service on port 25565 routing by hostname.
	// creative.mc.larnet.eu  →  minecraft-create-server-creative.minecraft-create.svc:25565
	// survival.mc.larnet.eu  →  minecraft-create-server-survival.minecraft-create.svc:25565
	router: {
		port:        25565
		serviceType: "LoadBalancer"
		// Players who connect without a matching hostname land on the creative server.
		defaultServer: {
			host: "minecraft-create-server-creative.minecraft-create.svc"
			port: 25565
		}
	}
}

// Values provide concrete configuration for the game-stack bundle.
// These satisfy the #config schema defined in bundle.cue.
//
// This example demonstrates both deployment modes simultaneously:
//   - Standalone servers: creative + minigames + create-mod (direct hostname per server)
//   - Proxied network:    lobby + survival behind a shared Velocity proxy (commented out)
package gamestack

values: {
	// Routing
	domain: "mc.example.com"
	// releaseName and namespace determine the in-cluster service hostnames:
	//   {releaseName}-{server}.{namespace}.svc
	releaseName: "my-game-stack"
	namespace:   "game-stack"

	// ── Standalone servers ────────────────────────────────────────────────────
	// Accessible directly via mc-router:
	//   creative.mc.example.com   →  my-game-stack-creative.game-stack.svc
	//   minigames.mc.example.com  →  my-game-stack-minigames.game-stack.svc
	//   create-mod.mc.example.com →  my-game-stack-create-mod.game-stack.svc
	servers: {

		// ── Creative server ───────────────────────────────────────────────────
		// Paper for plugin support (e.g. WorldEdit, CoreProtect).
		creative: {
			motd:       "Creative Mode — Build Freely"
			maxPlayers: 30
			mode:       "creative"
			pvp:        false
			difficulty: "peaceful"

			// Paper — high-performance Spigot fork with plugin ecosystem
			paper: {
				// plugins: {
				// 	// WorldEdit + CoreProtect for creative build management
				// 	modrinth: {
				// 		projects: ["worldedit", "coreprotect"]
				// 	}
				// }
			}

			storage: {
				data: {
					type: "pvc"
					size: "10Gi"
				}
				backups: {
					type: "pvc"
					size: "5Gi"
				}
			}
		}

		// // ── Minigames server ──────────────────────────────────────────────────
		// // Paper for its broad plugin ecosystem and Spiget integration.
		// minigames: {
		// 	motd:       "Minigames — Play and Compete!"
		// 	maxPlayers: 100
		// 	difficulty: "normal"

		// 	// Paper — best plugin compatibility for minigame frameworks
		// 	paper: {
		// 		plugins: {
		// 			// MiniGamesLib + EssentialsX via Spiget
		// 			spigetResources: [81534, 9089]
		// 		}
		// 	}

		// 	storage: {
		// 		data: {
		// 			type: "pvc"
		// 			size: "10Gi"
		// 		}
		// 		backups: {
		// 			type: "pvc"
		// 			size: "5Gi"
		// 		}
		// 	}
		// }

		// // ── Create mod server ─────────────────────────────────────────────────
		// // Forge 1.20.1 with the Create mod and a few common companions.
		// // Create is a tech/engineering mod — needs more RAM than vanilla.
		// "create-mod": {
		// 	version:    "1.20.1"
		// 	motd:       "Create Mod — Engineering Fun!"
		// 	maxPlayers: 20
		// 	mode:       "survival"
		// 	pvp:        false
		// 	difficulty: "easy"

		// 	// Forge — required by Create and its addon ecosystem
		// 	forge: {
		// 		version: "47.3.0"
		// 		mods: {
		// 			// Core mod + common companions via Modrinth
		// 			modrinth: {
		// 				projects: [
		// 					"create",          // Create: Mechanical engineering
		// 					"create-steam-n-rails", // Trains & railways addon
		// 					"jei",             // Just Enough Items — recipe viewer
		// 				]
		// 				downloadDependencies: "required"
		// 				allowedVersionType:   "release"
		// 			}
		// 		}
		// 	}

		// 	// Create is memory-heavy — give it more headroom
		// 	jvm: {
		// 		memory:       "4G"
		// 		useAikarFlags: true
		// 	}

		// 	storage: {
		// 		data: {
		// 			type: "pvc"
		// 			size: "10Gi"
		// 		}
		// 		backups: {
		// 			type: "pvc"
		// 			size: "5Gi"
		// 		}
		// 	}

		// 	// Enable tar backups for the modded world
		// 	backup: {
		// 		enabled:          true
		// 		interval:         "12h"
		// 		initialDelay:     "10m"
		// 		pruneBackupsDays: 14
		// 		pauseIfNoPlayers: true
		// 		tar: {
		// 			compressMethod: "zstd"
		// 			linkLatest:     true
		// 		}
		// 	}
		// }
	}

	// ── Proxied network ───────────────────────────────────────────────────────
	// Players connect at: play.mc.example.com → my-game-stack-proxy → lobby / survival
	network: {
		hostname: "play"

		proxy: {
			motd:             "Welcome to My Minecraft Network!"
			maxPlayers:       200
			forwardingMode:   "MODERN"
			forwardingSecret: "change-me-in-production"
		}

		servers: {
			lobby: {
				motd:       "Hub — Welcome!"
				maxPlayers: 100
				mode:       "adventure"
				pvp:        false
				difficulty: "peaceful"
				paper: {}
				storage: data: { type: "pvc", size: "10Gi" }
			}
			survival: {
				motd:       "Survival — Good Luck!"
				maxPlayers: 50
				difficulty: "hard"
				paper: {}
				storage: data: { type: "pvc", size: "20Gi" }
			}
		}
	}

	// RCON password applied to all Minecraft instances.
	// Use secretName+remoteKey to reference an existing K8s Secret:
	//   rconPassword: secretName: "mc-secrets", remoteKey: "rcon-pw"
	rconPassword: value: "change-me-in-production"
}

// Values provide concrete configuration for the Minecraft module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values - production-ready configuration
values: {
	// === Server Configuration ===
	server: {
		// Official itzg/minecraft-server image
		image: "itzg/minecraft-server:latest"

		// Paper: Optimized Minecraft server with plugin support
		// Best performance and most popular for multiplayer servers
		type: "PAPER"

		// Always download the latest stable version
		version: "LATEST"

		// EULA acceptance required by Mojang/Microsoft
		// Users must set this to true to deploy
		eula: true

		// Welcome message displayed to connecting players
		motd: "Welcome to OPM Minecraft Server!"

		// Maximum concurrent players
		maxPlayers: 20

		// Game settings
		difficulty: "normal"
		mode:       "survival"
		pvp:        true

		// Command blocks can be exploited - disabled by default
		enableCommandBlock: false

		// RCON configuration for backup coordination
		// IMPORTANT: Change this password in production!
		rcon: {
			password: "minecraft"
			port:     25575
		}
	}

	// === Storage Configuration ===
	storage: {
		// Game data: worlds, configs, plugins, player data
		// Worlds can grow large with exploration - 10Gi is a reasonable starting point
		data: {
			type: "pvc"
			size: "10Gi"
			// Optional: Specify storageClass for performance requirements
			// storageClass: "fast-ssd"
		}

		// Backup storage: Compressed backups accumulate over time
		// 20Gi allows ~2 weeks of daily backups with pruning enabled
		backups: {
			type: "pvc"
			size: "20Gi"
		}
	}

	// === Backup Configuration ===
	backup: {
		// Enable automated backups via sidecar container
		enabled: true

		// Official itzg/mc-backup image
		image: "itzg/mc-backup:latest"

		// Tar: Simple compressed backups, easy to restore
		method: "tar"

		// Backup once per day at the interval after initial delay
		interval: "24h"

		// Wait 5 minutes after server starts before first backup
		initialDelay: "5m"

		// Delete backups older than 7 days (keeps 1 week of daily backups)
		pruneBackupsDays: 7

		// Pause backups if server is empty (optional, conserves resources)
		pauseIfNoPlayers: false

		// Tar-specific settings
		tar: {
			compressMethod: "gzip" // Good balance of speed and compression
			linkLatest:     true   // Create symlink to latest backup for easy access
		}
	}

	// === Resource Limits ===
	// Minecraft is Java-based and can be memory-intensive
	resources: {
		requests: {
			cpu:    "1000m" // 1 CPU core minimum
			memory: "2Gi"   // 2GB RAM minimum for smooth gameplay
		}
		limits: {
			cpu:    "4000m" // Allow bursting to 4 cores during world generation
			memory: "8Gi"   // 8GB max - prevents OOM on busy servers
		}
	}

	// === Networking ===
	// Standard Minecraft server port
	port: 25565

	// LoadBalancer: Exposes server with external IP (cloud environments)
	// Change to NodePort for bare-metal or ClusterIP for internal-only
	serviceType: "LoadBalancer"
}

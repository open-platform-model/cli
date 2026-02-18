// Example: Production Server with Restic Cloud Backups to S3
// Usage: opm mod build examples/minecraft -f values-production.cue
package main

values: {
	server: {
		image:   "itzg/minecraft-server:latest"
		type:    "PAPER"
		version: "1.20.4"
		eula:    true
		motd:    "Production SMP Server - Powered by OPM"

		maxPlayers: 100
		difficulty: "normal"
		mode:       "survival"
		pvp:        true

		// Optimize view distance for performance
		viewDistance: 10

		// Server operators (change these!)
		ops: ["admin1", "admin2"]

		// Whitelist for private servers (optional)
		// whitelist: ["player1", "player2", "player3"]

		enableCommandBlock: false

		// CRITICAL: Change this password in production!
		rcon: {
			password: "USE-A-SECURE-PASSWORD-HERE"
			port:     25575
		}
	}

	storage: {
		// High-performance cloud storage
		data: {
			type:         "pvc"
			size:         "50Gi"
			storageClass: "standard" // Adjust based on your cloud provider
		}
		// Smaller backup storage - restic deduplicates
		backups: {
			type:         "pvc"
			size:         "10Gi"
			storageClass: "standard"
		}
	}

	backup: {
		enabled: false
		image:   "itzg/mc-backup:latest"
		method:  "restic"

		// Backup every 6 hours
		interval:     "6h"
		initialDelay: "10m"

		// Pause backups when server is empty (saves resources)
		pauseIfNoPlayers: true

		restic: {
			// S3 bucket configuration
			// IMPORTANT: Set AWS credentials via environment variables:
			// AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
			repository: "s3:s3.amazonaws.com/my-minecraft-backups"

			// Strong password for restic repository
			password: "CHANGE-THIS-RESTIC-PASSWORD"

			// Retention policy: Keep daily, weekly, and monthly backups
			retention: "--keep-daily 7 --keep-weekly 4 --keep-monthly 6"

			// Hostname identifies this server in restic
			hostname: "production-smp"

			// Enable verbose logging for debugging
			verbose: false
		}
	}

	// Production-grade resources
	resources: {
		cpu: {
			request: "2000m"
			limit:   "8000m"
		}
		memory: {
			request: "4Gi"
			limit:   "16Gi"
		}
	}

	port: 25565

	// LoadBalancer for cloud deployments
	serviceType: "LoadBalancer"
}

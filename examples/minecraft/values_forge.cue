// Example: Modded Forge Server with Local Storage
// Usage: opm mod build examples/minecraft -f values-forge.cue
package main

values: {
	server: {
		image: "itzg/minecraft-server:latest"
		type:  "FORGE"
		// Specific Forge version for mod compatibility
		version: "1.20.1"
		eula:    true
		motd:    "Modded Forge Server - Join the adventure!"

		// More players for modded servers
		maxPlayers: 50
		difficulty: "normal"
		mode:       "survival"
		pvp:        true

		// Disable command blocks for security
		enableCommandBlock: false

		// IMPORTANT: Change this password!
		rcon: {
			password: "CHANGE-THIS-PASSWORD"
			port:     25575
		}
	}

	storage: {
		// Use hostPath for bare-metal deployments
		data: {
			type:         "hostPath"
			path:         "/mnt/minecraft/modded-server"
			hostPathType: "DirectoryOrCreate"
		}
		backups: {
			type:         "hostPath"
			path:         "/mnt/minecraft/backups"
			hostPathType: "DirectoryOrCreate"
		}
	}

	backup: {
		enabled: true
		image:   "itzg/mc-backup:latest"
		method:  "tar"

		// Backup twice daily
		interval:     "12h"
		initialDelay: "10m"

		// Keep 5 days of backups
		pruneBackupsDays: 5

		tar: {
			compressMethod: "zstd" // Fast compression for large mod packs
			linkLatest:     true
		}
	}

	// Modded servers need more memory
	resources: {
		requests: {
			cpu:    "2000m"
			memory: "4Gi"
		}
		limits: {
			cpu:    "6000m"
			memory: "12Gi" // Mods can be very memory-intensive
		}
	}

	port: 25565

	// NodePort for bare-metal deployments
	serviceType: "NodePort"
}

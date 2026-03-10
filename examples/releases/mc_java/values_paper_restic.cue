// Values for a production Paper server with restic cloud backups (PVC storage, LoadBalancer).
// Based on the default values.cue; see #config in module.cue for the full schema.
package mc_java

values: {
	// === Server Type ===
	paper: {}

	// Minecraft game version
	version: "1.20.4"

	// Pin to Java 21 LTS — latest tag pulls java25 (early access) which has unstable init scripts
	image: {
		repository: "itzg/minecraft-server"
		tag:        "java21"
		digest:     ""
	}

	// === Server Properties ===
	server: {
		motd:       "Production SMP Server"
		maxPlayers: 100
		ops: ["player1", "player2"]
	}

	// === RCON ===
	rcon: {
		enabled: true
		password: value: "USE-A-SECURE-PASSWORD-HERE"
		port: 25575
	}

	// === Storage ===
	// PVC with fast-ssd storage class for production I/O requirements
	storage: {
		data: {
			type:         "pvc"
			size:         "50Gi"
			storageClass: "fast-ssd"
		}
		backups: {type: "pvc", size: "10Gi"}
	}

	// === Backup ===
	// Restic: encrypted, deduplicated, incremental backups to S3
	backup: {
		enabled:      true
		interval:     "6h"
		initialDelay: "10m"

		restic: {
			repository: "s3:s3.amazonaws.com/my-minecraft-backups"
			password: value: "restic-repo-password"
			hostname:  "production-smp"
			retention: "--keep-daily 7 --keep-weekly 4 --keep-monthly 6"
		}
	}

	// === Resources ===
	resources: {
		requests: {
			cpu:    "2000m"
			memory: "4Gi"
		}
		limits: {
			cpu:    "8000m"
			memory: "16Gi"
		}
	}

	// === Query ===
	query: port: 25565

	// === Networking ===
	port:        25565
	serviceType: "LoadBalancer"
}

// Values for a modded Forge server on bare-metal (hostPath storage, NodePort).
// Based on the default values.cue; see #config in module.cue for the full schema.
package main

values: {
	// === Server Type ===
	forge: {version: "47.2.0"}

	// Minecraft game version
	version: "1.20.1"

	// Pin to Java 21 LTS — latest tag pulls java25 (early access) which has unstable init scripts
	image: {
		repository: "itzg/minecraft-server"
		tag:        "java21"
		digest:     ""
	}

	// === JVM Configuration ===
	jvm: {
		memory: "8G"
		opts:   "-XX:+UseG1GC -XX:+ParallelRefProcEnabled"
	}

	// === Server Properties ===
	server: maxPlayers: 50

	// === RCON ===
	rcon: {
		enabled: true
		password: value: "CHANGE-THIS-PASSWORD"
		port: 25575
	}

	// === Storage ===
	// hostPath — suited for bare-metal nodes with local disks
	storage: {
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

	// === Backup ===
	// Disabled by default; tar is the default method
	backup: {
		enabled: false
		tar: {}
	}

	// === Resources ===
	resources: {
		requests: {
			cpu:    "2000m"
			memory: "8Gi"
		}
		limits: {
			cpu:    "4000m"
			memory: "12Gi"
		}
	}

	// === Query ===
	query: port: 25565

	// === Networking ===
	// NodePort must be in the valid range 30000-32767.
	// Clients connect to <node-ip>:30565 to reach the Minecraft server.
	port:        30565
	serviceType: "NodePort"
}

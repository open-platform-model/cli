// Values provide concrete configuration for the Minecraft module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values - testing configuration with ephemeral storage
values: {
	// === Server Type ===
	// Choose exactly one server type:
	paper: {}                           // ← active (auto-downloads latest Paper)
	// forge: { version: "47.2.0" }     // Forge alternative
	// fabric: { loaderVersion: "0.15.0" }
	// vanilla: {}

	// Pin to Java 21 LTS — latest tag pulls java25 (early access) which has unstable init scripts
	image: {
		repository: "itzg/minecraft-server"
		tag:        "java21"
		digest:     ""
	}

	// === RCON ===
	rcon: {
		enabled:  true
		password: value: "minecraft"
		port: 25575
	}

	// === Storage ===
	storage: {
		data: type:    "emptyDir"
		backups: type: "emptyDir"
	}

	// === Backup ===
	// Disabled by default; method/interval/initialDelay use schema defaults (tar/24h/5m)
	backup: enabled: false

	// === Resources ===
	resources: {
		requests: {
			cpu:    "500m"
			memory: "1Gi"
		}
		limits: {
			cpu:    "2000m"
			memory: "4Gi"
		}
	}

	// === Query ===
	query: port: 25565

	// === Networking ===
	port:        25565
	serviceType: "ClusterIP"
}

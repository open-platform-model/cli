// Values for a Fabric server with Modrinth performance mods (PVC storage, LoadBalancer).
// Based on the default values.cue; see #config in module.cue for the full schema.
//
// Note: In the #config schema, mods are nested inside the server type struct
// (fabric: { mods: {...} }), not at the top level.
package mc_java

values: {
	// === Server Type ===
	fabric: {
		loaderVersion: "0.15.0"
		mods: modrinth: {
			projects: ["sodium", "lithium", "starlight"]
			downloadDependencies: "required"
			allowedVersionType:   "release"
		}
	}

	// Minecraft game version
	version: "1.20.1"

	// Pin to Java 21 LTS — latest tag pulls java25 (early access) which has unstable init scripts
	image: {
		repository: "itzg/minecraft-server"
		tag:        "java21"
		digest:     ""
	}

	// === JVM Configuration ===
	jvm: memory: "6G"

	// === Server Properties ===
	server: motd: "Modrinth Modpack Server"

	// === RCON ===
	rcon: {
		enabled: true
		password: value: "CHANGE-ME"
		port: 25575
	}

	// === Storage ===
	// PVC — suited for cloud environments with dynamic provisioning
	storage: {
		data: {type: "pvc", size: "20Gi"}
		backups: {type: "pvc", size: "10Gi"}
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
			memory: "6Gi"
		}
		limits: {
			cpu:    "4000m"
			memory: "8Gi"
		}
	}

	// === Query ===
	query: port: 25565

	// === Networking ===
	port:        25565
	serviceType: "LoadBalancer"
}

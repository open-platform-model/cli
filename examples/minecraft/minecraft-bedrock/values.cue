// Values provide concrete configuration for the Minecraft Bedrock module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values - production-ready configuration
values: {
	server: {
		image: {
			repository: "itzg/minecraft-bedrock-server"
			tag:        "latest"
			digest:     ""
		}
		version:           "LATEST"
		eula:              true
		difficulty:        "easy"
		gameMode:          "survival"
		maxPlayers:        10
		defaultPermission: "member"
		serverName:        "Dedicated Server"
		onlineMode:        true
		maxThreads:        8
		cheats:            false
		serverPort:        19132
	}
	storage: data: {
		type: "pvc"
		size: "1Gi"
	}
	resources: {
		requests: {
			cpu:    "500m"
			memory: "512Mi"
		}
	}
	serviceType: "ClusterIP"
}

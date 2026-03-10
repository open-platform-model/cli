package mc_bedrock

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
		serverName:        "OPM Bedrock Server"
		onlineMode:        true
		maxThreads:        8
		cheats:            false
		serverPort:        19132
	}
	storage: data: {
		type: "pvc"
		size: "5Gi"
	}
	resources: {
		requests: {
			cpu:    "500m"
			memory: "512Mi"
		}
	}
	serviceType: "LoadBalancer"
}

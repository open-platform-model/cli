package rcon_web_admin

values: {
	admin: {
		isAdmin:      true
		username:     "admin"
		password:     value: "change-me"
		game:         "minecraft"
		serverName:   "Minecraft Java"
		rconHost:     "mc-java-server.default.svc"
		rconPort:     25575
		rconPassword: value: "change-me"
	}
	httpPort:    8080
	wsPort:      4327
	serviceType: "ClusterIP"
	resources: {
		requests: {
			cpu:    "50m"
			memory: "128Mi"
		}
	}
	securityContext: {
		runAsNonRoot:             false
		readOnlyRootFilesystem:   false
		allowPrivilegeEscalation: false
		capabilities: drop: ["ALL"]
	}
}

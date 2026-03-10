package mc_router

values: {
	router: {
		debug: false
		defaultServer: {
			host: "mc-java-server.default.svc"
			port: 25565
		}
		mappings: [{
			externalHostname: "play.example.com"
			host:             "mc-java-server.default.svc"
			port:             25565
		}]
		api: {
			enabled: false
			port:    8080
		}
	}
	port:        25565
	serviceType: "LoadBalancer"
	resources: {
		requests: {cpu: "50m", memory: "64Mi"}
		limits: {cpu: "200m", memory: "128Mi"}
	}
}

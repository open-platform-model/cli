package mc_monitor

values: {
	prometheus: {
		port: 8080
	}
	javaServers: [{
		host: "mc-java-server.default.svc"
	}]
	timeout: "1m0s"
	serviceType: "ClusterIP"
	resources: {
		requests: {
			cpu:    "50m"
			memory: "64Mi"
		}
		limits: {
			cpu:    "200m"
			memory: "128Mi"
		}
	}
}

package main

values: {
	router: {
		connectionRateLimit: 1
		debug:               true
		defaultServer: {
			host: "10.10.10.10"
			port: 25565
		}
		mappings: [
			{
				externalHostname: "play.example.com"
				host:             "10.10.10.10"
				port:             25565
			},
		]
	}
	port: 25565
}

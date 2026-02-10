package testmodule

// Module metadata
metadata: {
	name:             "test-module"
	version:          "1.0.0"
	fqn:              "example.com/test-module@v0#test-module"
	defaultNamespace: "default"
	labels: {
		"module.opmodel.dev/name":    metadata.name
		"module.opmodel.dev/version": metadata.version
	}
}

// Configuration schema
#config: {
	image:    string
	replicas: int & >=1
	port:     int | *8080
	debug:    bool | *false
}

// Concrete values
values: {
	image:    "nginx:1.25"
	replicas: 2
	port:     8080
	debug:    false
}

// Component definitions
#components: {
	// Web frontend component
	web: {
		metadata: {
			name: "web"
			labels: {
				"workload-type": "stateless"
			}
			annotations: {
				"description": "Web frontend"
			}
		}
		#resources: {
			"opmodel.dev/resources/Container@v0": {
				image:    #config.image
				replicas: #config.replicas
			}
		}
		#traits: {
			"opmodel.dev/traits/Expose@v0": {
				port: #config.port
			}
		}
		spec: {
			container: {
				image: #config.image
			}
			replicas: #config.replicas
		}
	}

	// API backend component
	api: {
		metadata: {
			name: "api"
			labels: {
				"workload-type": "stateless"
			}
		}
		#resources: {
			"opmodel.dev/resources/Container@v0": {
				image: "api:latest"
			}
		}
		spec: {
			container: {
				image: "api:latest"
			}
			replicas: 1
		}
	}

	// Worker component
	worker: {
		metadata: {
			name: "worker"
			labels: {
				"workload-type": "worker"
			}
		}
		#resources: {
			"opmodel.dev/resources/Container@v0": {
				image: "worker:latest"
			}
		}
		spec: {
			container: {
				image: "worker:latest"
			}
			replicas: 1
		}
	}
}

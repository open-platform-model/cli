package testmodule

// Module metadata
metadata: {
	name:             "test-module-no-values"
	version:          "1.0.0"
	fqn:              "example.com/test-module-no-values@v0#test-module-no-values"
	defaultNamespace: "default"
	labels: {
		"module.opmodel.dev/name":    metadata.name
		"module.opmodel.dev/version": metadata.version
	}
}

// Configuration schema — no defaults, must be provided via --values flag
#config: {
	image:    string
	replicas: int & >=1
	port:     int | *8080
}

// Component definitions
#components: {
	web: {
		metadata: {
			name: "web"
			labels: {
				"workload-type": "stateless"
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
}

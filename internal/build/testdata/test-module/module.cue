package testmodule

// Module metadata
metadata: {
	name:             "test-module"
	version:          "1.0.0"
	fqn:              "example.com/test-module@v0#test-module"
	uuid:             "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
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
}

// Concrete values
values: {
	image:    "nginx:1.25"
	replicas: 2
	port:     8080
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

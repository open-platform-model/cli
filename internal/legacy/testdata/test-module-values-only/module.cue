package testmodule

// Module metadata
metadata: {
	name:             "test-module-values-only"
	version:          "1.0.0"
	fqn:              "example.com/test-module-values-only@v0#test-module-values-only"
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

// NOTE: values are defined ONLY in values.cue (not here) to support
// clean override via --values flag.

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

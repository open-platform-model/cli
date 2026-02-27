package testmodule

// Module metadata (v1alpha1 format)
metadata: {
	modulePath:       "example.com/modules"
	name:             "test-module"
	version:          "1.0.0"
	fqn:              "example.com/modules/test-module:1.0.0"
	uuid:             "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	defaultNamespace: "default"
	labels: {
		"module.opmodel.dev/name":    metadata.name
		"module.opmodel.dev/version": metadata.version
	}
}

// Configuration schema
#config: {
	image: {
		repository: string
		tag:        string
		digest:     string
	}
	replicas: int & >=1
	port:     int | *8080
}

// Component definitions
#components: {
	web: {
		metadata: {
			name: "web"
			labels: {
				"core.opmodel.dev/workload-type": "stateless"
			}
		}
		#resources: {
			"opmodel.dev/resources/workload/container@v1": {
				image: #config.image
				scaling: count: #config.replicas
			}
		}
		#traits: {
			"opmodel.dev/traits/network/expose@v1": {
				port: #config.port
			}
		}
		spec: {
			container: {
				image: #config.image
			}
			scaling: count: #config.replicas
		}
	}
}

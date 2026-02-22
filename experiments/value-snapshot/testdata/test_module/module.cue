package testmodule

// metadata mirrors the real OPM module metadata shape.
// uuid is a static string here — in production it is computed by CUE via uid.SHA1,
// but the read path (Decision 9) is what we're proving: that these fields are
// accessible as concrete strings on the evaluated cue.Value.
metadata: {
	name:             "test-module"
	version:          "1.0.0"
	fqn:              "example.com/test-module@v0#TestModule"
	defaultNamespace: "default"
	uuid:             "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	labels: {
		"module.opmodel.dev/name":    metadata.name
		"module.opmodel.dev/version": metadata.version
	}
}

// #config is the user-facing configuration schema.
// Fields reference #config — at schema level they are constraints, not values.
#config: {
	image:    string | *"nginx:latest"
	replicas: int & >=1 | *1
	port:     int | *8080
	debug:    bool | *false
}

// values: #config — concrete defaults live in values.cue (loaded separately via
// Approach A). The module package itself carries only the abstract constraint.
values: #config

// #components defines the component schema.
// At Load() time these are schema-level: spec.image is `string`, not "nginx:latest".
// After FillPath("#config", userValues) they become concrete.
#components: {
	// web component — has both #resources and #traits
	web: {
		metadata: {
			name: "web"
			labels: {
				"workload-type": "stateless"
			}
		}
		#resources: {
			"example.com/Container@v0": {
				image:    #config.image
				replicas: #config.replicas
			}
		}
		#traits: {
			"example.com/Expose@v0": {
				port: #config.port
			}
		}
		spec: {
			image:    #config.image
			replicas: #config.replicas
			port:     #config.port
		}
	}

	// worker component — has #resources only (no #traits), tests optional traits
	worker: {
		metadata: {
			name: "worker"
			labels: {
				"workload-type": "worker"
			}
		}
		#resources: {
			"example.com/Container@v0": {
				image:    #config.image
				replicas: 1
			}
		}
		spec: {
			image:    #config.image
			replicas: 1
		}
	}
}

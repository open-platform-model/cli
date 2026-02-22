package patternbmodule

// pattern-b-module: Pattern B — inline values, no values.cue.
// Proves that the loader correctly populates mod.Values from mod.Raw
// when no separate values.cue exists in the module directory.

metadata: {
	name:             "pattern-b-module"
	version:          "1.0.0"
	fqn:              "example.com/pattern-b-module@v0#pattern-b-module"
	defaultNamespace: "default"
}

// Configuration schema
#config: {
	image:    string
	replicas: int & >=1
}

// Inline values — the sole defaults source (Pattern B).
// No values.cue exists alongside this file.
values: {
	image:    "nginx:stable"
	replicas: 2
}

#components: {
	web: {
		metadata: {
			name: "web"
			labels: "component.opmodel.dev/name": "web"
		}
		#resources: {
			"opmodel.dev/resources/Container@v0": {
				image:    #config.image
				replicas: #config.replicas
			}
		}
		#traits: {}
		spec: {
			image:    #config.image
			replicas: #config.replicas
		}
	}
}

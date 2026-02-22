// values_module/module.cue — pattern A: separate values.cue
//
// This module deliberately has NO values field and NO "values: #config".
// Concrete defaults live exclusively in values.cue, which is loaded separately
// by Approach A. The module package itself carries only the abstract #config schema.
package valuesmodule

apiVersion: "opmodel.dev/core/v0"
kind:       "Module"

metadata: {
	apiVersion: "example.com/values-module@v0"
	name:       "values-module"
	version:    "1.0.0"
}

// Schema only — constraints, no defaults.
// Concrete values never appear here; they live in values.cue.
#config: {
	image:    string
	replicas: int & >=1
}

#components: {
	web: {
		metadata: name: "web"
		spec: {
			image:    #config.image
			replicas: #config.replicas
		}
	}
}

// Package expmodule is a fake OPM module that mimics the shape of #Module
// WITHOUT importing opmodel.dev/core@v0. Used by Strategy A (dual-load) tests
// to probe whether a structurally-compatible cue.Value can be injected into
// #ModuleRelease.#module!: #Module when the catalog is loaded separately.
//
// The values here are carefully chosen so that CUE's computed fields would agree:
//   fqn  = "example.com/exp@v0#ExpModule"  (apiVersion + KebabToPascal(name))
//   uuid = SHA1(OPMNamespace, "example.com/exp@v0#ExpModule:0.1.0")
//
// Whether CUE accepts this value structurally (without core.#Module applied)
// is exactly what Decision 3 tests discover.
package expmodule

// Top-level fields matching #Module's required shape
apiVersion: "opmodel.dev/core/v0"
kind:       "Module"

metadata: {
	// Required fields that match #Module.metadata constraints
	apiVersion: "example.com/exp@v0"
	name:       "exp-module"
	version:    "0.1.0"
	// fqn and uuid are NOT declared here — they are computed fields in #Module.
	// Decision 3 will discover what CUE does when they are absent from the injected value.
}

// Configuration schema — matches the constraint shape in #Module
#config: {
	image:    string | *"nginx:latest"
	replicas: int & >=1 | *1
}

// Concrete default values
values: {
	image:    "nginx:latest"
	replicas: 1
}

// Components — structured like #Component but without the type constraint applied
#components: {
	web: {
		metadata: {
			name: "web"
			labels: "workload-type": "stateless"
		}
		spec: {
			image:    #config.image
			replicas: #config.replicas
		}
	}
}

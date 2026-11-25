// Vet fixture on the current schema line (opmodel.dev/core@v2). A structurally
// valid #Module that defines #config with a default for every field, pins no
// catalog and has no components: `opm module vet` passes its identity and
// #config checks and renders zero objects against a platform generated from
// its own deps, the verdict build reaches for the same input.
package simple_module

import m "opmodel.dev/core@v2"

m.#Module

metadata: {
	name:       "simple_module"
	modulePath: "example.com/modules/simple_module@v0"
	version:    "0.1.0"
}

// Configuration schema with defaults.
#config: {
	replicas: *1 | int
	image:    *"nginx:latest" | string
}

// Empty debugValues: every field takes its #config default. Core's #Module
// leaves debugValues open, and synthesis refuses an open values source, so a
// module relying on its defaults declares an empty one.
debugValues: {}

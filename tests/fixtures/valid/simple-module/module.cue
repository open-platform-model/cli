// Vet fixture on the current schema line (opmodel.dev/core@v2). A structurally
// valid #Module that defines #config with a default for every field and
// deliberately no debugValues of its own: core's #Module leaves debugValues
// open, and `opm module vet` merges that open value with #config to a
// concrete result through the kernel, so the module passes — the verdict
// build reaches for the same input.
package simple_module

import m "opmodel.dev/core@v2"

m.#Module

metadata: {
	name:       "simple_module"
	modulePath: "example.com/modules/simple_module@v0"
	version:    "0.1.0"
}

// Configuration schema with defaults. No debugValues field: the vet test
// asserts the open debugValues merge to a concrete value through the defaults.
#config: {
	replicas: *1 | int
	image:    *"nginx:latest" | string
}

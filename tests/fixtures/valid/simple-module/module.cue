// Vet fixture on the current schema line (opmodel.dev/core@v2). A structurally
// valid #Module that defines #config but deliberately no debugValues, so
// `opm module vet` reaches and fails the debugValues check — the same behavior
// this fixture had on the retired v1alpha1 line.
package simple_module

import m "opmodel.dev/core@v2"

m.#Module

metadata: {
	name:       "simple_module"
	modulePath: "example.com/modules/simple_module@v0"
	version:    "0.1.0"
}

// Configuration schema with defaults. No debugValues field: the vet test
// asserts the module is rejected for not defining one.
#config: {
	replicas: *1 | int
	image:    *"nginx:latest" | string
}

// Vet fixture on the current schema line (opmodel.dev/core@v1). A structurally
// valid #Module that defines #config but deliberately no debugValues, so
// `opm module vet` reaches and fails the debugValues check — the same behavior
// this fixture had on the retired v1alpha1 line.
package simplemodule

import m "opmodel.dev/core@v1"

m.#Module

metadata: {
	modulePath: "example.com/modules"
	name:       "simple-module"
	version:    "0.1.0"
}

// Configuration schema with defaults. No debugValues field: the vet test
// asserts the module is rejected for not defining one.
#config: {
	replicas: *1 | int
	image:    *"nginx:latest" | string
}

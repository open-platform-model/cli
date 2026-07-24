// Vet fixture on the current schema line (opmodel.dev/core@v1) exercising the
// debugValues path: a valid #Module whose debugValues make #config concrete, so
// module vet passes. Ported from the retired v1alpha1 line (which carried only a
// cue.mod — this restores the module body its name promises).
package modulewithdebugvalues

import m "opmodel.dev/core@v1"

m.#Module

metadata: {
	modulePath: "example.com/modules"
	name:       "module-with-debug-values"
	version:    "0.1.0"
}

#config: {
	replicas: int & >=1 | *1
	image:    string | *"nginx:latest"
}

debugValues: {
	replicas: 2
	image:    "nginx:1.28"
}

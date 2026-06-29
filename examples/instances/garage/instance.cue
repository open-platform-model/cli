// Garage instance example.
// Imports the public garage module (opmodel.dev/modules/garage@v0)
// and binds it to a ModuleInstance.
//
// Build:   opm instance build ./examples/instances/garage/instance.cue
// Apply:   opm instance apply ./examples/instances/garage/instance.cue --create-namespace
package garage

import (
	mr "opmodel.dev/core/v1alpha1/modulerelease@v1"
	m "opmodel.dev/modules/garage@v0"
)

mr.#ModuleInstance

metadata: {
	name:      "garage"
	namespace: "garage"
}

#module: m

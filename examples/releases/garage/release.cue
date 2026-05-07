// Garage release example.
// Imports the public garage module (opmodel.dev/modules/garage@v0)
// and binds it to a ModuleRelease.
//
// Build:   opm release build ./examples/releases/garage/release.cue
// Apply:   opm release apply ./examples/releases/garage/release.cue --create-namespace
package garage

import (
	mr "opmodel.dev/core/v1alpha1/modulerelease@v1"
	m "opmodel.dev/modules/garage@v0"
)

mr.#ModuleRelease

metadata: {
	name:      "garage"
	namespace: "garage"
}

#module: m

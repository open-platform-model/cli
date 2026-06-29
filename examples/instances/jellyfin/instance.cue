// Jellyfin instance example.
// Imports the public jellyfin module from the OPM catalog
// (opmodel.dev/modules/jellyfin@v1) and binds it to a ModuleInstance.
//
// Build:   opm instance build ./examples/instances/jellyfin/instance.cue
// Apply:   opm instance apply ./examples/instances/jellyfin/instance.cue --create-namespace
package jellyfin

import (
	mr "opmodel.dev/core/v1alpha1/modulerelease@v1"
	m "opmodel.dev/modules/jellyfin@v1"
)

mr.#ModuleInstance

metadata: {
	name:      "jellyfin"
	namespace: "default"
}

#module: m

// Jellyfin release example.
// Imports the public jellyfin module from the OPM catalog
// (opmodel.dev/modules/jellyfin@v1) and binds it to a ModuleRelease.
//
// Build:   opm release build ./examples/releases/jellyfin/release.cue
// Apply:   opm release apply ./examples/releases/jellyfin/release.cue --create-namespace
package jellyfin

import (
	mr "opmodel.dev/core/v1alpha1/modulerelease@v1"
	m "opmodel.dev/modules/jellyfin@v1"
)

mr.#ModuleRelease

metadata: {
	name:      "jellyfin"
	namespace: "default"
}

#module: m

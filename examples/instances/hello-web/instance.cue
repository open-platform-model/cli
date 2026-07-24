// hello-web instance example (opmodel.dev/core@v1).
// Imports the neutral hello-web test module
// (opmodel.dev/modules/test/hello-web@v0) and binds it to a #ModuleInstance.
// Renders a single Deployment.
//
// Build:   opm instance build ./examples/instances/hello-web/instance.cue
// Apply:   opm instance apply ./examples/instances/hello-web/instance.cue --create-namespace
package hello_web

import (
	core "opmodel.dev/core@v1"
	// The module's CUE package is hello_web (underscore); the import path's last
	// element hello-web is not a valid CUE identifier, so name the package explicitly.
	helloweb "opmodel.dev/modules/test/hello-web@v0:hello_web"
)

core.#ModuleInstance

metadata: {
	name:      "hello-web"
	namespace: "default"
}

#module: helloweb

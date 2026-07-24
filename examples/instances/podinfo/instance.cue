// podinfo instance example (opmodel.dev/core@v1).
// Imports the neutral podinfo test module
// (opmodel.dev/modules/test/podinfo@v0) and binds it to a #ModuleInstance.
// Renders a Deployment + Service with HTTP liveness/readiness probes.
//
// Build:   opm instance build ./examples/instances/podinfo/instance.cue
// Apply:   opm instance apply ./examples/instances/podinfo/instance.cue --create-namespace
package podinfo

import (
	core "opmodel.dev/core@v1"
	podinfo "opmodel.dev/modules/test/podinfo@v0"
)

core.#ModuleInstance

metadata: {
	name:      "podinfo"
	namespace: "default"
}

#module: podinfo

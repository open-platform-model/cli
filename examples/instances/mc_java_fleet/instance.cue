// Minecraft Java fleet instance example.
// Imports the public mc_java_fleet module
// (opmodel.dev/modules/mc_java_fleet@v0) and binds it to a ModuleInstance.
//
// The fleet module wraps multiple Minecraft Java servers behind a shared
// mc-router that does hostname-based routing. The default values.cue
// defines a single 'survival' server; values_multi.cue demonstrates a
// two-server fleet with the router exposed as LoadBalancer.
//
// Build (single server):
//   opm instance build ./examples/instances/mc_java_fleet/instance.cue
// Build (multi-server):
//   opm instance build ./examples/instances/mc_java_fleet/instance.cue \
//     -f ./examples/instances/mc_java_fleet/values_multi.cue
package mc_java_fleet

import (
	mr "opmodel.dev/core/v1alpha1/modulerelease@v1"
	m "opmodel.dev/modules/mc_java_fleet@v0"
)

mr.#ModuleInstance

metadata: {
	name:      "mc-java-fleet"
	namespace: "default"
}

#module: m

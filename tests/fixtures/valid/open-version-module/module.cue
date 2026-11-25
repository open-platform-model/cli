// Vet fixture with an open identity Version. metadata derives modulePath and
// version from the identity package, so metadata.version is open too. The
// module defines #config with a default for every field, pins no catalog and
// has no components: `opm module vet` passes its identity, coordinate and
// #config checks, then skips the render because the version is open.
package open_version_module

import (
	m "opmodel.dev/core@v2"

	id "example.com/modules/open_version_module/identity"
)

m.#Module

metadata: {
	name:       "open_version_module"
	modulePath: id.ModulePath
	version:    id.Version
}

// Configuration schema with defaults.
#config: {
	replicas: *1 | int
	image:    *"nginx:latest" | string
}

// Empty debugValues: every field takes its #config default.
debugValues: {}

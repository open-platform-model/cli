package jellyfin

import (
	mr "opmodel.dev/core/v1alpha1/modulerelease@v1"
	jf "opmodel.dev/examples/modules/jellyfin@v1"
)

// Module definition
mr.#ModuleRelease

metadata: {
	name:      "jf"
	namespace: "jellyfin"
}

#module: jf

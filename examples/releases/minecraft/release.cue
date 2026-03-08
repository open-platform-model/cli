package minecraft

import (
	mr "opmodel.dev/core/modulerelease@v1"
	mc "opmodel.dev/examples/modules/mc_java@v1"
)

// Module definition
mr.#ModuleRelease

metadata: {
	name:      "minecraft"
	namespace: "minecraft"
}

#module: mc

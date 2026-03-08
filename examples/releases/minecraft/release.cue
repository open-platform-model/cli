package minecraft

import (
	mr "opmodel.dev/core/modulerelease@v1"
	mc "opmodel.dev/examples/modules/minecraft@v1"
)

// Module definition
mr.#ModuleRelease

metadata: {
	name:      "minecraft-java-release"
	namespace: "minecraft"
}

#module: mc

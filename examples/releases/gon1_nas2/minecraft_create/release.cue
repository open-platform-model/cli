package minecraft_create

import (
	mr    "opmodel.dev/core/modulerelease@v1"
	fleet "opmodel.dev/examples/modules/mc_java_fleet@v1"
)

mr.#ModuleRelease

metadata: {
	name:      "minecraft-create"
	namespace: "minecraft-create"
}

#module: fleet

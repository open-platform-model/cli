package mc_velocity

import (
	mr "opmodel.dev/core/modulerelease@v1"
	mod "opmodel.dev/examples/modules/mc_velocity@v1"
)

mr.#ModuleRelease

metadata: {
	name:      "mc-velocity"
	namespace: "default"
}

#module: mod

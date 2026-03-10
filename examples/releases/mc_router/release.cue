package mc_router

import (
	mr "opmodel.dev/core/modulerelease@v1"
	mod "opmodel.dev/examples/modules/mc_router@v1"
)

mr.#ModuleRelease

metadata: {
	name:      "mc-router"
	namespace: "default"
}

#module: mod

package main

import (
	mr "opmodel.dev/core/modulerelease@v1"
	zot "opmodel.dev/examples/modules/zot_registry@v1"
)

mr.#ModuleRelease

metadata: {
	name:      "registry"
	namespace: "zot-registry"
}

#module: zot

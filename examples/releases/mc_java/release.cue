package mc_java

import (
	mr "opmodel.dev/core/v1alpha1/modulerelease@v1"
	mc "opmodel.dev/examples/modules/mc_java@v1"
)

mr.#ModuleRelease

metadata: {
	name:      "mc-java"
	namespace: "default"
}

#module: mc

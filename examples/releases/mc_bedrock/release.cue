package mc_bedrock

import (
	mr "opmodel.dev/core/v1alpha1/modulerelease@v1"
	mod "opmodel.dev/examples/modules/mc_bedrock@v1"
)

mr.#ModuleRelease

metadata: {
	name:      "mc-bedrock"
	namespace: "default"
}

#module: mod

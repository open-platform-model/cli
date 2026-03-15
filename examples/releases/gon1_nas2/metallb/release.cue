package metallb

import (
	mr      "opmodel.dev/core/modulerelease@v1"
	metallb "opmodel.dev/examples/modules/metallb@v1"
)

mr.#ModuleRelease

metadata: {
	name:      "metallb"
	namespace: "metallb-system"
}

#module: metallb

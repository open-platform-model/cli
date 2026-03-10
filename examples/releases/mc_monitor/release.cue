package mc_monitor

import (
	mr "opmodel.dev/core/modulerelease@v1"
	mod "opmodel.dev/examples/modules/mc_monitor@v1"
)

mr.#ModuleRelease

metadata: {
	name:      "mc-monitor"
	namespace: "default"
}

#module: mod

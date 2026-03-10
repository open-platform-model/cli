package rcon_web_admin

import (
	mr "opmodel.dev/core/modulerelease@v1"
	mod "opmodel.dev/examples/modules/rcon_web_admin@v1"
)

mr.#ModuleRelease

metadata: {
	name:      "rcon-web-admin"
	namespace: "default"
}

#module: mod

// Instance of the backup_demo test module, in the module's own CUE module:
// it imports the module package by its path, and renders against the same
// dependency pins.
package instance

import (
	core "opmodel.dev/core@v2"
	demo "example.com/modules/backup_demo@v0"
)

core.#ModuleInstance

metadata: {
	name:      "backup-demo"
	namespace: "default"
}

#module: demo

values: {}

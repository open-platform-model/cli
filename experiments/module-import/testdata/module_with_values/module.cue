// Package withvalues: Tests whether values.cue breaks importability
package withvalues

import (
	"opmodel.dev/core@v1"
)

// Embed #Module at package root
core.#Module

metadata: {
	modulePath: "test.dev/modules"
	name:       "withvalues"
	version:    "0.1.0"
	description: "Module with values.cue to test conflicts"
}

#config: {
	image: {
		repository: string | *"nginx"
		tag:        string | *"latest"
	}
	replicas: int | *1
}

// Package simple: A minimal flattened OPM module for import testing.
// This tests whether embedding core.#Module at package root works for imports.
package simple

import (
	"opmodel.dev/core@v1"
)

// Embed #Module at package root (flattened style)
core.#Module

// Module metadata
metadata: {
	modulePath: "test.dev/modules"
	name:       "simple"
	version:    "0.1.0"
	description: "A minimal test module"
}

// Schema with defaults
#config: {
	image: {
		repository: string | *"nginx"
		tag:        string | *"latest"
	}
	replicas: int | *1
}

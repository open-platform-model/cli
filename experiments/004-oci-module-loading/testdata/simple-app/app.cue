package app

import (
	core "opm.dev/core@v0"
)

// Define a simple module
module: core.#Module & {
	metadata: {
		name:        "simple-app"
		version:     "1.0.0"
		description: "A simple test application"
	}

	components: {
		web: {
			metadata: {
				name: "web"
				labels: {
					"core.opm.dev/workload-type": "stateless"
				}
			}
		}
	}
}

// Create a release
release: core.#ModuleRelease & {
	#module: module
	metadata: {
		name:      "simple-app"
		namespace: "default"
		version:   module.metadata.version
	}
}

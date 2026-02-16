// Package main defines the multi-package module.
// This example demonstrates organizing components in separate files.
// In a real multi-package setup, components would be in a separate directory,
// but CUE requires complex module configuration for that to work.
package main

import (
	"opmodel.dev/core@v0"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	apiVersion:       "example.com/multi-package-module@v0"
	name:             "multi-package-module"
	version:          "0.1.0"
	description:      string | *"Multi-file module organization example"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// Frontend configuration
	frontend: {
		image:    string
		replicas: int & >=1
		port:     int & >0 & <=65535
	}

	// Backend configuration
	backend: {
		image:    string
		replicas: int & >=1
		port:     int & >0 & <=65535
	}

	// Worker configuration
	worker: {
		image:    string
		replicas: int & >=1
	}
}

// Components defined in separate files (frontend.cue, backend.cue, worker.cue)
// #components is populated by those files

// Values must satisfy #config - concrete values in values.cue
values: #config

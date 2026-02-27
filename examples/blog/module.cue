// Package main defines the blog module.
// This is a standard module with separated concerns:
// - module.cue: metadata and schema
// - components.cue: component definitions
// - values.cue: default values
package main

import (
	"opmodel.dev/core@v1"
	schemas "opmodel.dev/schemas@v1"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	modulePath:       "example.com/modules"
	name:             "blog"
	version:          "0.1.0"
	description:      string | *"A standard OPM module"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// Web component configuration
	web: {
		image:    schemas.#Image
		replicas: int & >=1
		port:     int & >0 & <=65535
	}

	// API component configuration
	api: {
		image:    schemas.#Image
		replicas: int & >=1
		port:     int & >0 & <=65535
	}
}


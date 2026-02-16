// Package main defines the blog module.
// This is a standard module with separated concerns:
// - module.cue: metadata and schema
// - components.cue: component definitions
// - values.cue: default values
package main

import (
	"opmodel.dev/core@v0"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	apiVersion:       "example.com/blog@v0"
	name:             "blog"
	version:          "0.1.0"
	description:      string | *"A standard OPM module"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// Web component configuration
	web: {
		image:    string
		replicas: int & >=1
		port:     int & >0 & <=65535
	}

	// API component configuration
	api: {
		image:    string
		replicas: int & >=1
		port:     int & >0 & <=65535
	}
}

// Values must satisfy #config - concrete values in values.cue
values: #config

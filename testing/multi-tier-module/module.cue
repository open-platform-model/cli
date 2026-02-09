// Package main defines the multi-tier module.
// This is a complex module demonstrating all workload types:
// - module.cue: metadata and schema
// - components.cue: component definitions (stateful, daemon, task, scheduled-task)
// - values.cue: default values
package main

import (
	"opmodel.dev/core@v0"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	apiVersion:       "example.com/multi-tier-module@v0"
	name:             "multi-tier-module"
	version:          "0.1.0"
	description:      string | *"A multi-tier OPM module with all workload types"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// Database component configuration (stateful workload)
	database: {
		image:   string
		scaling: int & >=1
	}

	// Log agent component configuration (daemon workload)
	logAgent: {
		image: string
	}

	// Setup job component configuration (task workload)
	setupJob: {
		image: string
	}

	// Backup job component configuration (scheduled-task workload)
	backupJob: {
		image:    string
		schedule: string
	}
}

// Values must satisfy #config - concrete values in values.cue
values: #config

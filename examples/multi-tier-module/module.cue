// Package main defines the multi-tier module.
// This is a complex module demonstrating all workload types:
// - module.cue: metadata and schema
// - components.cue: component definitions (stateful, daemon, task, scheduled-task)
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
	name:             "multi-tier-module"
	version:          "0.1.0"
	description:      string | *"A multi-tier OPM module with all workload types"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// Database component configuration (stateful workload)
	database: {
		image:   schemas.#Image
		scaling: int & >=1
	}

	// Log agent component configuration (daemon workload)
	logAgent: {
		image: schemas.#Image
	}

	// Setup job component configuration (task workload)
	setupJob: {
		image: schemas.#Image
	}

	// Backup job component configuration (scheduled-task workload)
	backupJob: {
		image:    schemas.#Image
		schedule: string
	}
}


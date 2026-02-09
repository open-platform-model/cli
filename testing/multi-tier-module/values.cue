// Values provide concrete configuration for the module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values
values: {
	// Database (stateful workload) configuration
	database: {
		image:   "postgres:14"
		scaling: 1
	}

	// Log agent (daemon workload) configuration
	logAgent: {
		image: "prom/node-exporter:v1.6.1"
	}

	// Setup job (task workload) configuration
	setupJob: {
		image: "myregistry.io/migrations:v2.0.0"
	}

	// Backup job (scheduled-task workload) configuration
	backupJob: {
		image:    "postgres:14"
		schedule: "0 2 * * *"
	}
}

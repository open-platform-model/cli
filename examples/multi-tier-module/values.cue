// Values provide concrete configuration for the module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values
values: {
	// Database (stateful workload) configuration
	database: {
		image: {
			repository: "postgres"
			tag:        "14"
			digest:     ""
		}
		scaling: 1
	}

	// Log agent (daemon workload) configuration
	logAgent: {
		image: {
			repository: "prom/node-exporter"
			tag:        "v1.6.1"
			digest:     ""
		}
	}

	// Setup job (task workload) configuration
	setupJob: {
		image: {
			repository: "myregistry.io/migrations"
			tag:        "v2.0.0"
			digest:     ""
		}
	}

	// Backup job (scheduled-task workload) configuration
	backupJob: {
		image: {
			repository: "postgres"
			tag:        "14"
			digest:     ""
		}
		schedule: "0 2 * * *"
	}
}

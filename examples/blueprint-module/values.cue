// Values provide concrete configuration for the module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values
values: {
	// API component
	api: {
		image: {
			repository: "node"
			tag:        "20-alpine"
			digest:     ""
		}
		replicas: 3
		port:     3000
	}

	// Database component (PostgreSQL)
	database: {
		image: {
			repository: "postgres"
			tag:        "15-alpine"
			digest:     ""
		}
		engine:   "postgres"
		dbName:   "myapp"
		username: "appuser"
		password: "change-me-in-production"
		storage: {
			size:         "10Gi"
			storageClass: "standard"
		}
	}
}

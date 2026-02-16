// Values provide concrete configuration for the module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values
values: {
	// API component
	api: {
		image:    "node:20-alpine"
		replicas: 3
		port:     3000
	}

	// Database component (PostgreSQL)
	database: {
		engine:   "postgres"
		version:  "15-alpine"
		dbName:   "myapp"
		username: "appuser"
		password: "change-me-in-production"
		storage: {
			size:         "10Gi"
			storageClass: "standard"
		}
	}
}

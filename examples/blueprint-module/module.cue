// Package main defines the blueprint-module example.
// Demonstrates a two-tier module with a stateless API and a stateful database.
// Note: Blueprints (opmodel.dev/blueprints) are not yet available in v1alpha1.
// This example uses raw resources + traits directly, which is the equivalent approach.
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
	name:             "blueprint-module"
	version:          "0.1.0"
	description:      string | *"Two-tier module: stateless API + stateful database"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// API component (stateless workload)
	api: {
		image:    schemas.#Image
		replicas: int & >=1
		port:     int & >0 & <=65535
	}

	// Database component (stateful workload)
	database: {
		image:    schemas.#Image
		engine:   "postgres" | "mysql" | "mongodb" | "redis"
		dbName:   string
		username: string
		password: string
		storage: {
			size:         string
			storageClass: string | *"standard"
		}
	}
}


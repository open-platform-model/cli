// Package main defines the blueprint-module example.
// Demonstrates "easy mode" module authoring using blueprints.
// Blueprints eliminate boilerplate by pre-composing resources + traits.
package main

import (
	"opmodel.dev/core@v0"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	apiVersion:       "example.com/blueprint-module@v0"
	name:             "blueprint-module"
	version:          "0.1.0"
	description:      string | *"Blueprint-based module authoring example"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// API component (stateless workload)
	api: {
		image:    string
		replicas: int & >=1
		port:     int & >0 & <=65535
	}

	// Database component (simple database)
	database: {
		engine:   "postgres" | "mysql" | "mongodb" | "redis"
		version:  string
		dbName:   string
		username: string
		password: string
		storage: {
			size:         string
			storageClass: string | *"standard"
		}
	}
}

// Values must satisfy #config - concrete values in values.cue
values: #config

// Values provide concrete configuration for the module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values
values: {
	// Web frontend configuration
	web: {
		image:    "nginx:1.25"
		replicas: 4
		port:     8080
	}

	// API backend configuration
	api: {
		image:    "node:20-alpine"
		replicas: 4
		port:     3000
	}
}

// Values provide concrete configuration for the module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values
values: {
	// Web frontend configuration
	web: {
		image: {
			repository: "nginx"
			tag:        "latest"
			digest:     ""
		}
		replicas: 4
		port:     8080
	}

	// API backend configuration
	api: {
		image: {
			repository: "node"
			tag:        "20-alpine"
			digest:     ""
		}
		replicas: 4
		port:     3000
	}
}

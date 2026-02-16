// Values provide concrete configuration for the module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values
values: {
	frontend: {
		image:    "nginx:1.25-alpine"
		replicas: 3
		port:     8080
	}

	backend: {
		image:    "node:20-alpine"
		replicas: 3
		port:     3000
	}

	worker: {
		image:    "python:3.11-slim"
		replicas: 2
	}
}

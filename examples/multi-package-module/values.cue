// Values provide concrete configuration for the module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values
values: {
	frontend: {
		image: {
			repository: "nginx"
			tag:        "1.25-alpine"
			digest:     ""
		}
		replicas: 3
		port:     8080
	}

	backend: {
		image: {
			repository: "node"
			tag:        "20-alpine"
			digest:     ""
		}
		replicas: 3
		port:     3000
	}

	worker: {
		image: {
			repository: "python"
			tag:        "3.11-slim"
			digest:     ""
		}
		replicas: 2
	}
}

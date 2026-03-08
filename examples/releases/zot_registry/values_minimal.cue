package main

// Minimal configuration for development and CI environments
// No authentication, no persistence, debug logging
values: {
	image: {
		variant:    "minimal"
		tag:        "v2.1.14"
		digest:     ""
		pullPolicy: "IfNotPresent"
	}

	storage: {
		type:    "emptyDir"
		rootDir: "/var/lib/registry"
		dedupe:  false
	}

	http: {
		port:    5000
		address: "0.0.0.0"
	}

	log: {
		level: "debug"
	}

	// No authentication in minimal mode

	// No metrics in minimal mode

	replicas: 1

	resources: {
		requests: {
			memory: "128Mi"
			cpu:    "100m"
		}
		limits: {
			memory: "512Mi"
			cpu:    "250m"
		}
	}

	security: {
		runAsNonRoot:             true
		runAsUser:                1000
		runAsGroup:               1000
		readOnlyRootFilesystem:   false
		allowPrivilegeEscalation: false
		capabilities: {
			drop: ["ALL"]
		}
	}
}

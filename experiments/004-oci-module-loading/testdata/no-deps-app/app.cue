package app

// Self-contained module without external dependencies
// This tests the loader without needing a registry

module: {
	metadata: {
		name:        "no-deps-app"
		version:     "1.0.0"
		description: "Self-contained test module"
	}

	components: {
		web: {
			metadata: {
				name: "web"
				labels: {
					"app": "nginx"
				}
			}

			spec: {
				image:    "nginx:latest"
				replicas: 3
			}
		}
	}
}

// Metadata for the test
info: {
	loaded:    true
	timestamp: "2026-01-28"
}

// Staging environment values.
// Override development defaults with staging-specific configuration.
// Build with: opm mod build -f values_staging.cue ./examples/values-layering
package main

// Staging overrides
values: {
	environment: "staging"

	web: {
		// Staging uses versioned image (not 'latest')
		image: "nginx:1.25.3-alpine"

		// Staging scales up for load testing
		replicas: 3

		port: 8080

		// Higher resource requests for realistic load
		resources: {
			requests: {
				cpu:    "100m"
				memory: "128Mi"
			}
			limits: {
				cpu:    "500m"
				memory: "256Mi"
			}
		}

		// Staging ingress with TLS
		ingress: {
			hostname: "webapp-staging.example.com"
			tls: {
				enabled:    true
				secretName: "webapp-staging-tls"
			}
		}
	}
}

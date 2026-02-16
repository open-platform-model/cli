// Base values for DEVELOPMENT environment.
// These provide sane defaults optimized for local development.
// Override with values_staging.cue or values_production.cue for other environments.
package main

// Development defaults
values: {
	environment: "dev"

	web: {
		// Dev uses lightweight image
		image: "nginx:1.25-alpine"

		// Dev uses minimal resources
		replicas: 1

		port: 8080

		// Low resource requests for local development
		resources: {
			requests: {
				cpu:    "50m"
				memory: "64Mi"
			}
			limits: {
				cpu:    "200m"
				memory: "128Mi"
			}
		}

		// Dev ingress (no TLS)
		ingress: {
			hostname: "webapp-dev.local"
			path:     "/"
			tls: {
				enabled: false
			}
		}
	}
}

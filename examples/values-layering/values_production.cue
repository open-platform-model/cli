// Production environment values.
// Override development defaults with production-specific configuration.
// Build with: opm mod build -f values_production.cue ./examples/values-layering
package main

// Production overrides
values: {
	environment: "production"

	web: {
		// Production uses pinned SHA digest for immutability
		image: "nginx:1.25.3-alpine@sha256:a59278fd22a9d411121e190b8cec8aa57b306aa3332459197777583beb728f59"

		// Production requires high availability (enforced by schema constraint)
		replicas: 5

		port: 8080

		// Production-grade resource allocation
		resources: {
			requests: {
				cpu:    "200m"
				memory: "256Mi"
			}
			limits: {
				cpu:    "1000m"
				memory: "512Mi"
			}
		}

		// Production ingress with TLS (enforced by schema constraint)
		ingress: {
			hostname: "webapp.example.com"
			tls: {
				enabled:    true
				secretName: "webapp-production-tls"
			}
		}
	}
}

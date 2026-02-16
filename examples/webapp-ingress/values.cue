// Values provide concrete configuration for the module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values
values: {
	web: {
		// Container image
		image: "nginx:1.25-alpine"

		// Service port
		port: 8080

		// Autoscaling: min 2, max 10 replicas, scale at 70% CPU
		scaling: {
			min:                  2
			max:                  10
			targetCPUUtilization: 70
		}

		// Resource requests and limits
		resources: {
			requests: {
				cpu:    "100m"
				memory: "128Mi"
			}
			limits: {
				cpu:    "500m"
				memory: "512Mi"
			}
		}

		// Ingress configuration
		ingress: {
			hostname:         "webapp.example.com"
			path:             "/"
			ingressClassName: "nginx"
			tls: {
				enabled:    false
				secretName: "webapp-tls"
			}
		}

		// Security: run as non-root user
		security: {
			runAsUser:  1000
			runAsGroup: 1000
			fsGroup:    1000
		}

		// Service account
		serviceAccount: {
			name: "webapp"
		}

		// Sidecar: disabled by default
		sidecar: {
			enabled: false
			image:   "fluent/fluent-bit:2.0-distroless"
		}
	}
}

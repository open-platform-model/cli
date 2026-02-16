// Package main defines the webapp-ingress module.
// A production-grade web application demonstrating:
// - HTTP Ingress routing
// - Horizontal Pod Autoscaling (HPA)
// - Security hardening (SecurityContext)
// - Service account management (WorkloadIdentity)
// - Sidecar containers
package main

import (
	"opmodel.dev/core@v0"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	apiVersion:       "example.com/webapp-ingress@v0"
	name:             "webapp-ingress"
	version:          "0.1.0"
	description:      string | *"Production web app with Ingress, HPA, and security"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// Web application configuration
	web: {
		// Container image
		image: string

		// Service port
		port: int & >0 & <=65535

		// Autoscaling configuration
		scaling: {
			min: int & >=1 & <=100
			max: int & >=1 & <=100
			targetCPUUtilization: int & >0 & <=100
		}

		// Resource requests and limits
		resources: {
			requests: {
				cpu:    string
				memory: string
			}
			limits: {
				cpu:    string
				memory: string
			}
		}

		// Ingress configuration
		ingress: {
			hostname:         string
			path:             string | *"/"
			ingressClassName: string | *"nginx"
			tls: {
				enabled:     bool | *false
				secretName?: string
			}
		}

		// Security configuration
		security: {
			runAsUser:  int | *1000
			runAsGroup: int | *1000
			fsGroup:    int | *1000
		}

		// Service account
		serviceAccount: {
			name: string
		}

		// Sidecar configuration
		sidecar: {
			enabled: bool | *false
			image:   string
		}
	}
}

// Values must satisfy #config - concrete values in values.cue
values: #config

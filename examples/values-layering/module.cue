// Package main defines the values-layering module.
// Demonstrates environment-specific configuration using values override files.
// Base values for development, with staging and production overrides.
package main

import (
	"opmodel.dev/core@v0"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	apiVersion:       "example.com/values-layering@v0"
	name:             "values-layering"
	version:          "0.1.0"
	description:      string | *"Environment-specific configuration example"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// Environment label (validated)
	environment: "dev" | "staging" | "production"

	// Web application configuration
	web: {
		image:    string
		replicas: int & >=1 & <=100
		port:     int & >0 & <=65535

		// Resource configuration
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
			hostname: string
			path:     string | *"/"
			tls: {
				enabled:     bool | *false
				secretName?: string
			}
		}

		// Environment-specific constraints
		if environment == "production" {
			// Production requires at least 2 replicas
			replicas: >=2
		}

		if environment == "production" {
			// Production requires TLS
			ingress: tls: enabled: true
		}
	}
}

// Values must satisfy #config - concrete values in values.cue
values: #config

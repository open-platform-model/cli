// Package main defines the app-config module.
// Demonstrates externalized configuration using ConfigMaps and Secrets:
// - ConfigMap for application settings
// - Secrets for credentials and sensitive data
// - Volume-mounted config files
// - Environment variable wiring from config
package main

import (
	"opmodel.dev/core@v1"
	schemas "opmodel.dev/schemas@v1"
)

// Module definition
core.#Module

// Module metadata
metadata: {
	modulePath:       "example.com/modules"
	name:             "app-config"
	version:          "0.1.0"
	description:      string | *"Application with ConfigMaps and Secrets"
	defaultNamespace: "default"
}

// Schema only - constraints for users, no defaults
#config: {
	// Application configuration
	app: {
		image:    schemas.#Image
		port:     int & >0 & <=65535
		replicas: int & >=1

		// Application settings (stored in ConfigMap)
		settings: {
			logLevel:       string
			maxConnections: int & >0
			timeout:        string
			cacheEnabled:   bool
		}

		// Database credentials (stored in Secret)
		database: {
			host:     string
			port:     int & >0 & <=65535
			name:     string
			username: string
			password: string // Sensitive
		}

		// API keys (stored in Secret)
		apiKeys: {
			github:  string
			slack:   string
			datadog: string
		}

		// Config file content (mounted as volume)
		configFile: {
			fileName: string
			content:  string
		}
	}
}


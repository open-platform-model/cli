// Values provide concrete configuration for the module.
// These satisfy the #config schema defined in module.cue.
package main

// Concrete default values
values: {
	app: {
		image:    "node:20-alpine"
		port:     3000
		replicas: 2

		// Application settings
		settings: {
			logLevel:       "info"
			maxConnections: 100
			timeout:        "30s"
			cacheEnabled:   true
		}

		// Database credentials
		database: {
			host:     "postgres.database.svc.cluster.local"
			port:     5432
			name:     "myapp"
			username: "appuser"
			password: "change-me-in-production" // Should be overridden
		}

		// API keys
		apiKeys: {
			github:  "ghp_example_token_replace_in_prod"
			slack:   "https://hooks.slack.com/services/EXAMPLE"
			datadog: "dd_api_key_replace_in_prod"
		}

		// Config file content
		configFile: {
			fileName: "app.yaml"
			content: """
				server:
				  port: 3000
				  host: 0.0.0.0
				
				database:
				  pool_size: 10
				  timeout: 5000
				
				cache:
				  ttl: 3600
				  max_size: 1000
				
				logging:
				  format: json
				  timestamp: true
				"""
		}
	}
}

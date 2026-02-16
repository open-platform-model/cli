// Components defines the application workload with config and secrets.
// Demonstrates ConfigMaps, Secrets, and volume-mounted configuration.
package main

import (
	"encoding/base64"
	resources_workload "opmodel.dev/resources/workload@v0"
	resources_config "opmodel.dev/resources/config@v0"
	resources_storage "opmodel.dev/resources/storage@v0"
	traits_workload "opmodel.dev/traits/workload@v0"
	traits_network "opmodel.dev/traits/network@v0"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	/////////////////////////////////////////////////////////////////
	//// App - Stateless Application with Config and Secrets
	/////////////////////////////////////////////////////////////////

	app: {
		resources_workload.#Container
		resources_config.#ConfigMaps
		resources_config.#Secrets
		resources_storage.#Volumes
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_network.#Expose

		metadata: {
			name: "app"
			labels: "core.opmodel.dev/workload-type": "stateless"
		}

		spec: {
			scaling: count: #config.app.replicas

			restartPolicy: "Always"

			// ConfigMaps for application settings
			configMaps: {
				"app-settings": {
					data: {
						"log_level":        #config.app.settings.logLevel
						"max_connections":  "\(#config.app.settings.maxConnections)"
						"timeout":          #config.app.settings.timeout
						"cache_enabled":    "\(#config.app.settings.cacheEnabled)"
					}
				}

				"app-config-file": {
					data: {
						(#config.app.configFile.fileName): #config.app.configFile.content
					}
				}
			}

			// Secrets for credentials
			secrets: {
				"db-credentials": {
					type: "Opaque"
					data: {
						"host":     base64.Encode(null, #config.app.database.host)
						"port":     base64.Encode(null, "\(#config.app.database.port)")
						"database": base64.Encode(null, #config.app.database.name)
						"username": base64.Encode(null, #config.app.database.username)
						"password": base64.Encode(null, #config.app.database.password)
					}
				}

				"api-keys": {
					type: "Opaque"
					data: {
						"github":  base64.Encode(null, #config.app.apiKeys.github)
						"slack":   base64.Encode(null, #config.app.apiKeys.slack)
						"datadog": base64.Encode(null, #config.app.apiKeys.datadog)
					}
				}
			}

			// Volumes for config file
			volumes: {
				"config-file": {
					name: "config-file"
					configMap: {
						// ConfigMap volumes reference the ConfigMap by the volume name
						// The transformer will map this to the actual ConfigMap resource
					}
				}
			}

			// Container definition
			container: {
				name:  "app"
				image: #config.app.image
				ports: http: {
					name:       "http"
					targetPort: #config.app.port
					protocol:   "TCP"
				}

				// Environment variables from ConfigMap
				env: {
					PORT: {
						name:  "PORT"
						value: "\(#config.app.port)"
					}
					LOG_LEVEL: {
						name:  "LOG_LEVEL"
						value: #config.app.settings.logLevel
					}
					MAX_CONNECTIONS: {
						name:  "MAX_CONNECTIONS"
						value: "\(#config.app.settings.maxConnections)"
					}
					TIMEOUT: {
						name:  "TIMEOUT"
						value: #config.app.settings.timeout
					}
					CACHE_ENABLED: {
						name:  "CACHE_ENABLED"
						value: "\(#config.app.settings.cacheEnabled)"
					}

					// Environment variables from Secrets
					DB_HOST: {
						name:  "DB_HOST"
						value: #config.app.database.host
					}
					DB_PORT: {
						name:  "DB_PORT"
						value: "\(#config.app.database.port)"
					}
					DB_NAME: {
						name:  "DB_NAME"
						value: #config.app.database.name
					}
					DB_USERNAME: {
						name:  "DB_USERNAME"
						value: #config.app.database.username
					}
					DB_PASSWORD: {
						name:  "DB_PASSWORD"
						value: #config.app.database.password
					}

					// API keys from Secret
					GITHUB_API_KEY: {
						name:  "GITHUB_API_KEY"
						value: #config.app.apiKeys.github
					}
					SLACK_WEBHOOK_URL: {
						name:  "SLACK_WEBHOOK_URL"
						value: #config.app.apiKeys.slack
					}
					DATADOG_API_KEY: {
						name:  "DATADOG_API_KEY"
						value: #config.app.apiKeys.datadog
					}

					// Config file path
					CONFIG_FILE_PATH: {
						name:  "CONFIG_FILE_PATH"
						value: "/etc/app/\(#config.app.configFile.fileName)"
					}
				}

				// Mount config file as volume
				volumeMounts: {
					"config-file": {
						name:      "config-file"
						mountPath: "/etc/app"
						readOnly:  true
					}
				}
			}

			// Service exposure
			expose: {
				ports: http: container.ports.http & {
					exposedPort: #config.app.port
				}
				type: "ClusterIP"
			}
		}
	}
}

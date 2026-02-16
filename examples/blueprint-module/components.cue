// Components using blueprints for simplified authoring.
// Compare this to manual resource+trait composition in other examples.
package main

import (
	blueprints_workload "opmodel.dev/blueprints/workload@v0"
	blueprints_data "opmodel.dev/blueprints/data@v0"
	traits_network "opmodel.dev/traits/network@v0"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	/////////////////////////////////////////////////////////////////
	//// API - Stateless Workload Blueprint
	/////////////////////////////////////////////////////////////////

	api: {
		// Use the StatelessWorkload blueprint
		blueprints_workload.#StatelessWorkload

		// Also attach network traits (not in blueprint)
		traits_network.#Expose

		metadata: {
			name: "api"
			// No need to set workload-type label - blueprint sets it automatically
		}

		spec: {
			// Blueprint uses a single "statelessWorkload" field
			// This wraps all the underlying resources + traits
			statelessWorkload: {
				container: {
					name:  "api"
					image: #config.api.image
					ports: http: {
						name:       "http"
						targetPort: #config.api.port
						protocol:   "TCP"
					}
					env: {
						PORT: {
							name:  "PORT"
							value: "\(#config.api.port)"
						}
					}
				}

				scaling: {
					count: #config.api.replicas
				}

				restartPolicy: "Always"

				healthCheck: {
					livenessProbe: {
						httpGet: {
							path: "/healthz"
							port: #config.api.port
						}
						initialDelaySeconds: 30
						periodSeconds:       10
					}
					readinessProbe: {
						httpGet: {
							path: "/ready"
							port: #config.api.port
						}
						initialDelaySeconds: 10
						periodSeconds:       5
					}
				}
			}

			// Service exposure (separate trait, not in blueprint)
			expose: {
				ports: http: statelessWorkload.container.ports.http & {
					exposedPort: #config.api.port
				}
				type: "ClusterIP"
			}
		}
	}

	/////////////////////////////////////////////////////////////////
	//// Database - SimpleDatabase Blueprint
	/////////////////////////////////////////////////////////////////

	database: {
		// Use the SimpleDatabase blueprint
		blueprints_data.#SimpleDatabase

		metadata: {
			name: "database"
			// No need to set workload-type label - blueprint sets it automatically
		}

		spec: {
			// Blueprint uses a single "simpleDatabase" field
			// This auto-generates container, volumes, env vars, health checks
			simpleDatabase: {
				engine:   #config.database.engine
				version:  #config.database.version
				dbName:   #config.database.dbName
				username: #config.database.username
				password: #config.database.password

				persistence: {
					enabled:      true
					size:         #config.database.storage.size
					storageClass: #config.database.storage.storageClass
				}
			}
		}
	}
}

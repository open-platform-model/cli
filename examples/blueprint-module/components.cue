// Components using raw resources + traits for a two-tier application.
// Demonstrates stateless API + stateful database pattern.
package main

import (
	resources_workload "opmodel.dev/resources/workload@v1"
	resources_storage "opmodel.dev/resources/storage@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	/////////////////////////////////////////////////////////////////
	//// API - Stateless Workload
	/////////////////////////////////////////////////////////////////

	api: {
		resources_workload.#Container
		traits_workload.#Scaling
		traits_workload.#HealthCheck
		traits_workload.#RestartPolicy
		traits_network.#Expose

		metadata: {
			name: "api"
			labels: "core.opmodel.dev/workload-type": "stateless"
		}

		spec: {
			scaling: count: #config.api.replicas

			restartPolicy: "Always"

			healthCheck: {
				livenessProbe: {
					httpGet: {
						path: "/healthz"
						port: #config.api.port
					}
					initialDelaySeconds: 30
					periodSeconds:       10
					timeoutSeconds:      5
					failureThreshold:    3
				}
				readinessProbe: {
					httpGet: {
						path: "/ready"
						port: #config.api.port
					}
					initialDelaySeconds: 10
					periodSeconds:       5
					timeoutSeconds:      3
					failureThreshold:    3
				}
			}

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
					DB_HOST: {
						name:  "DB_HOST"
						value: "database"
					}
					DB_NAME: {
						name:  "DB_NAME"
						value: #config.database.dbName
					}
					DB_USER: {
						name:  "DB_USER"
						value: #config.database.username
					}
					DB_PASSWORD: {
						name:  "DB_PASSWORD"
						value: #config.database.password
					}
				}
			}

			expose: {
				ports: http: container.ports.http & {
					exposedPort: #config.api.port
				}
				type: "ClusterIP"
			}
		}
	}

	/////////////////////////////////////////////////////////////////
	//// Database - Stateful Workload
	/////////////////////////////////////////////////////////////////

	database: {
		resources_workload.#Container
		resources_storage.#Volumes
		traits_workload.#Scaling
		traits_workload.#HealthCheck
		traits_workload.#RestartPolicy

		metadata: {
			name: "database"
			labels: "core.opmodel.dev/workload-type": "stateful"
		}

		spec: {
			scaling: count: 1

			restartPolicy: "Always"

			healthCheck: {
				livenessProbe: {
					exec: command: ["pg_isready", "-U", #config.database.username]
					initialDelaySeconds: 30
					periodSeconds:       10
					timeoutSeconds:      5
					failureThreshold:    3
				}
				readinessProbe: {
					exec: command: ["pg_isready", "-U", #config.database.username]
					initialDelaySeconds: 5
					periodSeconds:       10
					timeoutSeconds:      1
					failureThreshold:    3
				}
			}

			container: {
				name:            "database"
				image:           #config.database.image
				imagePullPolicy: "IfNotPresent"
				ports: db: {
					name:       "db"
					targetPort: 5432
				}
				env: {
					POSTGRES_DB: {
						name:  "POSTGRES_DB"
						value: #config.database.dbName
					}
					POSTGRES_USER: {
						name:  "POSTGRES_USER"
						value: #config.database.username
					}
					POSTGRES_PASSWORD: {
						name:  "POSTGRES_PASSWORD"
						value: #config.database.password
					}
				}
				volumeMounts: data: {
					name:      "data"
					mountPath: "/var/lib/postgresql/data"
				}
			}

			volumes: data: {
				name: "data"
				persistentClaim: {
					size:         #config.database.storage.size
					storageClass: #config.database.storage.storageClass
				}
			}
		}
	}
}

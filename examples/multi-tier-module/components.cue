// Components defines the workloads for this module.
// Covers all workload types: stateful, daemon, task, scheduled-task.
package main

import (
	// "opmodel.dev/core@v0"
	resources_workload "opmodel.dev/resources/workload@v0"
	resources_storage "opmodel.dev/resources/storage@v0"
	traits_workload "opmodel.dev/traits/workload@v0"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	/////////////////////////////////////////////////////////////////
	//// Database - Stateful Workload
	/////////////////////////////////////////////////////////////////

	database: {
		resources_workload.#Container
		resources_storage.#Volumes
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_workload.#UpdateStrategy
		traits_workload.#HealthCheck
		traits_workload.#InitContainers

		metadata: name: "database"
		metadata: labels: "core.opmodel.dev/workload-type": "stateful"

		spec: {
			scaling: count: #config.database.scaling
			restartPolicy: "Always"
			updateStrategy: {
				type: "RollingUpdate"
				rollingUpdate: {
					maxUnavailable: 1
					partition:      0
				}
			}
			healthCheck: {
				livenessProbe: {
					exec: command: ["pg_isready", "-U", "admin"]
					initialDelaySeconds: 30
					periodSeconds:       10
					timeoutSeconds:      5
					failureThreshold:    3
				}
				readinessProbe: {
					exec: command: ["pg_isready", "-U", "admin"]
					initialDelaySeconds: 5
					periodSeconds:       10
					timeoutSeconds:      1
					failureThreshold:    3
				}
			}
			initContainers: [{
				name:  "init-db"
				image: #config.database.image
				env: PGHOST: {
					name:  "PGHOST"
					value: "localhost"
				}
			}]
			container: {
				name:            "postgres"
				image:           #config.database.image
				imagePullPolicy: "IfNotPresent"
				ports: postgres: {
					name:       "postgres"
					targetPort: 5432
				}
				env: {
					POSTGRES_DB: {
						name:  "POSTGRES_DB"
						value: "myapp"
					}
					POSTGRES_USER: {
						name:  "POSTGRES_USER"
						value: "admin"
					}
					POSTGRES_PASSWORD: {
						name:  "POSTGRES_PASSWORD"
						value: "secretpassword"
					}
				}
				resources: {
					requests: {
						cpu:    "500m"
						memory: "1Gi"
					}
					limits: {
						cpu:    "2000m"
						memory: "4Gi"
					}
				}
				volumeMounts: data: {
					name:      "data"
					mountPath: "/var/lib/postgresql/data"
				}
			}
			volumes: data: {
				name: "data"
				persistentClaim: size: "10Gi"
			}
		}
	}

	/////////////////////////////////////////////////////////////////
	//// Log Agent - Daemon Workload
	/////////////////////////////////////////////////////////////////

	"log-agent": {
		resources_workload.#Container
		traits_workload.#RestartPolicy
		traits_workload.#UpdateStrategy
		traits_workload.#HealthCheck

		metadata: name: "log-agent"
		metadata: labels: "core.opmodel.dev/workload-type": "daemon"

		spec: {
			restartPolicy: "Always"
			updateStrategy: {
				type: "RollingUpdate"
				rollingUpdate: maxUnavailable: 1
			}
			healthCheck: {
				livenessProbe: {
					httpGet: {
						path: "/metrics"
						port: 9100
					}
					initialDelaySeconds: 15
					periodSeconds:       20
				}
				readinessProbe: {
					httpGet: {
						path: "/metrics"
						port: 9100
					}
					initialDelaySeconds: 5
					periodSeconds:       10
				}
			}
			container: {
				name:            "node-exporter"
				image:           #config.logAgent.image
				imagePullPolicy: "IfNotPresent"
				ports: metrics: {
					name:       "metrics"
					targetPort: 9100
				}
				resources: {
					requests: {
						cpu:    "100m"
						memory: "128Mi"
					}
					limits: {
						cpu:    "200m"
						memory: "256Mi"
					}
				}
				volumeMounts: {
					proc: {
						name:      "proc"
						mountPath: "/host/proc"
						readOnly:  true
					}
					sys: {
						name:      "sys"
						mountPath: "/host/sys"
						readOnly:  true
					}
				}
			}
		}
	}

	/////////////////////////////////////////////////////////////////
	//// Setup Job - Task Workload
	/////////////////////////////////////////////////////////////////

	"setup-job": {
		resources_workload.#Container
		traits_workload.#RestartPolicy
		traits_workload.#JobConfig
		traits_workload.#InitContainers

		metadata: name: "setup-job"
		metadata: labels: "core.opmodel.dev/workload-type": "task"

		spec: {
			restartPolicy: "OnFailure"
			jobConfig: {
				completions:             1
				parallelism:             1
				backoffLimit:            3
				activeDeadlineSeconds:   3600
				ttlSecondsAfterFinished: 86400
			}
			initContainers: [{
				name:  "pre-migration-check"
				image: #config.setupJob.image
				env: CHECK_MODE: {
					name:  "CHECK_MODE"
					value: "true"
				}
			}]
			container: {
				name:            "migration"
				image:           #config.setupJob.image
				imagePullPolicy: "IfNotPresent"
				env: {
					DATABASE_URL: {
						name:  "DATABASE_URL"
						value: "postgres://localhost:5432/myapp"
					}
					MIGRATION_VERSION: {
						name:  "MIGRATION_VERSION"
						value: "v2.0.0"
					}
				}
				resources: {
					requests: {
						cpu:    "500m"
						memory: "512Mi"
					}
					limits: {
						cpu:    "1000m"
						memory: "1Gi"
					}
				}
			}
		}
	}

	/////////////////////////////////////////////////////////////////
	//// Backup Job - Scheduled Task Workload
	/////////////////////////////////////////////////////////////////

	"backup-job": {
		resources_workload.#Container
		traits_workload.#RestartPolicy
		traits_workload.#CronJobConfig
		traits_workload.#InitContainers

		metadata: name: "backup-job"
		metadata: labels: "core.opmodel.dev/workload-type": "scheduled-task"

		spec: {
			restartPolicy: "OnFailure"
			cronJobConfig: {
				scheduleCron:               #config.backupJob.schedule
				concurrencyPolicy:          "Forbid"
				startingDeadlineSeconds:    300
				successfulJobsHistoryLimit: 3
				failedJobsHistoryLimit:     1
			}
			initContainers: [{
				name:  "pre-backup-check"
				image: #config.backupJob.image
				env: PGHOST: {
					name:  "PGHOST"
					value: "postgres-service"
				}
			}]
			container: {
				name:            "backup"
				image:           #config.backupJob.image
				imagePullPolicy: "IfNotPresent"
				env: {
					PGHOST: {
						name:  "PGHOST"
						value: "postgres-service"
					}
					PGUSER: {
						name:  "PGUSER"
						value: "admin"
					}
					PGPASSWORD: {
						name:  "PGPASSWORD"
						value: "secretpassword"
					}
					BACKUP_LOCATION: {
						name:  "BACKUP_LOCATION"
						value: "/backups"
					}
				}
				resources: {
					requests: {
						cpu:    "250m"
						memory: "256Mi"
					}
					limits: {
						cpu:    "500m"
						memory: "512Mi"
					}
				}
				volumeMounts: backups: {
					name:      "backup-storage"
					mountPath: "/backups"
				}
			}
		}
	}
}

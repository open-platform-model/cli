package hybrid

import (
	workload_blueprints "test.com/experiment/pkg/blueprints/workload"
	data_blueprints "test.com/experiment/pkg/blueprints/data"
	networking_traits "test.com/experiment/pkg/traits/network"
)

/////////////////////////////////////////////////////////////////
//// Component 1: web — StatelessWorkload + Expose
/////////////////////////////////////////////////////////////////

webComponent: workload_blueprints.#StatelessWorkload & networking_traits.#Expose & {
	metadata: {
		name: "web"
		labels: {
			"core.opm.dev/workload-type":  "stateless"
			"app.kubernetes.io/component": "frontend"
		}
	}
	spec: {
		statelessWorkload: {
			container: {
				name:  "nginx"
				image: string
				ports: http: {
					name:       "http"
					targetPort: 80
				}
			}
			replicas:      int | *1
			restartPolicy: "Always"
		}
		expose: {
			type: "ClusterIP"
			ports: http: {
				name:        "http"
				targetPort:  80
				exposedPort: 80
			}
		}
	}
}

/////////////////////////////////////////////////////////////////
//// Component 2: api — StatelessWorkload + Expose
/////////////////////////////////////////////////////////////////

apiComponent: workload_blueprints.#StatelessWorkload & networking_traits.#Expose & {
	metadata: {
		name: "api"
		labels: {
			"core.opm.dev/workload-type":  "stateless"
			"app.kubernetes.io/component": "backend"
		}
	}
	spec: {
		statelessWorkload: {
			container: {
				name:  "api-server"
				image: string
				ports: api: {
					name:       "api"
					targetPort: 8080
				}
				env: {
					LOG_LEVEL: {
						name:  "LOG_LEVEL"
						value: "info"
					}
				}
			}
			replicas:      int | *1
			restartPolicy: "Always"
			healthCheck: {
				readinessProbe: {
					httpGet: {
						path: "/health"
						port: 8080
					}
					initialDelaySeconds: 5
					periodSeconds:       10
				}
			}
		}
		expose: {
			type: "ClusterIP"
			ports: api: {
				name:        "api"
				targetPort:  8080
				exposedPort: 8080
			}
		}
	}
}

/////////////////////////////////////////////////////////////////
//// Component 3: database — SimpleDatabase
/////////////////////////////////////////////////////////////////

databaseComponent: data_blueprints.#SimpleDatabase & {
	metadata: {
		name: "database"
		labels: {
			"core.opm.dev/workload-type":  "stateful"
			"app.kubernetes.io/component": "database"
		}
	}
	spec: {
		simpleDatabase: {
			engine:   "postgres"
			version:  string
			dbName:   "appdb"
			username: "admin"
			password: string
			persistence: {
				enabled: true
				size:    string
			}
		}
	}
}

/////////////////////////////////////////////////////////////////
//// Component 4: cache — StatefulWorkload
/////////////////////////////////////////////////////////////////

cacheComponent: workload_blueprints.#StatefulWorkload & {
	metadata: {
		name: "cache"
		labels: {
			"core.opm.dev/workload-type":  "stateful"
			"app.kubernetes.io/component": "cache"
		}
	}
	spec: {
		statefulWorkload: {
			container: {
				name:  "redis"
				image: string
				ports: redis: {
					name:       "redis"
					targetPort: 6379
				}
				volumeMounts: {
					data: {
						name:      "data"
						mountPath: "/data"
					}
				}
			}
			replicas:      1
			restartPolicy: "Always"
			volumes: {
				data: {
					name: "data"
					persistentClaim: {
						size:         string
						accessMode:   "ReadWriteOnce"
						storageClass: "standard"
					}
				}
			}
		}
	}
}

/////////////////////////////////////////////////////////////////
//// Component 5: logAgent — DaemonWorkload
/////////////////////////////////////////////////////////////////

logAgentComponent: workload_blueprints.#DaemonWorkload & {
	metadata: {
		name: "log-agent"
		labels: {
			"core.opm.dev/workload-type":  "daemon"
			"app.kubernetes.io/component": "logging"
		}
	}
	spec: {
		daemonWorkload: {
			container: {
				name:  "node-exporter"
				image: string
				ports: metrics: {
					name:       "metrics"
					targetPort: 9100
				}
			}
			restartPolicy: "Always"
			updateStrategy: {
				type: "RollingUpdate"
				rollingUpdate: {
					maxUnavailable: 1
				}
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
			}
		}
	}
}

/////////////////////////////////////////////////////////////////
//// Component 6: migration — TaskWorkload
/////////////////////////////////////////////////////////////////

migrationComponent: workload_blueprints.#TaskWorkload & {
	metadata: {
		name: "migration"
		labels: {
			"core.opm.dev/workload-type":  "task"
			"app.kubernetes.io/component": "migration"
		}
	}
	spec: {
		taskWorkload: {
			container: {
				name:  "migration-runner"
				image: string
				env: {
					DATABASE_URL: {
						name:  "DATABASE_URL"
						value: "postgres://admin:password@database:5432/appdb"
					}
					MIGRATION_VERSION: {
						name:  "MIGRATION_VERSION"
						value: string
					}
				}
			}
			restartPolicy: "OnFailure"
			jobConfig: {
				completions:             1
				parallelism:             1
				backoffLimit:            3
				activeDeadlineSeconds:   3600
				ttlSecondsAfterFinished: 86400
			}
		}
	}
}

/////////////////////////////////////////////////////////////////
//// Component 7: backup — ScheduledTaskWorkload
/////////////////////////////////////////////////////////////////

backupComponent: workload_blueprints.#ScheduledTaskWorkload & {
	metadata: {
		name: "backup"
		labels: {
			"core.opm.dev/workload-type":  "scheduled-task"
			"app.kubernetes.io/component": "backup"
		}
	}
	spec: {
		scheduledTaskWorkload: {
			container: {
				name:  "backup-runner"
				image: string
				env: {
					PGHOST: {
						name:  "PGHOST"
						value: "database"
					}
					PGUSER: {
						name:  "PGUSER"
						value: "admin"
					}
					PGPASSWORD: {
						name:  "PGPASSWORD"
						value: string
					}
					BACKUP_LOCATION: {
						name:  "BACKUP_LOCATION"
						value: "/backups"
					}
				}
			}
			restartPolicy: "OnFailure"
			cronJobConfig: {
				scheduleCron:               string
				concurrencyPolicy:          "Forbid"
				startingDeadlineSeconds:    300
				successfulJobsHistoryLimit: 3
				failedJobsHistoryLimit:     1
			}
		}
	}
}

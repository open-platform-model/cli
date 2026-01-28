package experiment

import (
	core "test.com/experiment/pkg/core"
)

allBlueprintsModule: core.#Module & {
	metadata: {
		apiVersion: "opm.dev@v0"
		name:       "AllBlueprintsModule"
		version:    "0.1.0"
		description: "Comprehensive test module using all 6 blueprint types"
	}

	#components: {
		// Component 1: web (StatelessWorkload + Expose)
		web: webComponent & {
			spec: {
				statelessWorkload: {
					replicas: values.web.replicas
					container: {
						image: values.web.image
					}
				}
				expose: {
					ports: http: {
						name:        "http"
						targetPort:  80
						exposedPort: 80
					}
				}
			}
		}

		// Component 2: api (StatelessWorkload, no Expose)
		api: apiComponent & {
			spec: {
				statelessWorkload: {
					replicas: values.api.replicas
					container: {
						image: values.api.image
					}
				}
			}
		}

		// Component 3: database (SimpleDatabase)
		database: databaseComponent & {
			spec: simpleDatabase: {
				version:  values.database.version
				password: values.database.password
				persistence: size: values.database.volumeSize
			}
		}

		// Component 4: cache (StatefulWorkload)
		cache: cacheComponent & {
			spec: {
				statefulWorkload: {
					container: {
						image: values.cache.image
					}
					volumes: data: {
						persistentClaim: {
							size: values.cache.volumeSize
						}
					}
				}
			}
		}

		// Component 5: logAgent (DaemonWorkload)
		logAgent: logAgentComponent & {
			spec: {
				daemonWorkload: {
					container: {
						image: values.logAgent.image
					}
				}
			}
		}

		// Component 6: migration (TaskWorkload)
		migration: migrationComponent & {
			spec: {
				taskWorkload: {
					container: {
						image: values.migration.image
						env: MIGRATION_VERSION: value: values.migration.version
					}
				}
			}
		}

		// Component 7: backup (ScheduledTaskWorkload)
		backup: backupComponent & {
			spec: {
				scheduledTaskWorkload: {
					container: {
						image: values.backup.image
						env: PGPASSWORD: value: values.backup.password
					}
					cronJobConfig: {
						scheduleCron: values.backup.schedule
					}
				}
			}
		}
	}

	config: {
		web: {
			replicas: int
			image:    string
		}
		api: {
			replicas: int
			image:    string
		}
		database: {
			version:    string
			password:   string
			volumeSize: string
		}
		cache: {
			image:      string
			volumeSize: string
		}
		logAgent: {
			image: string
		}
		migration: {
			image:   string
			version: string
		}
		backup: {
			image:    string
			password: string
			schedule: string
		}
	}

	values: {
		web: {
			replicas: 2
			image:    "nginx:1.21.0"
		}
		api: {
			replicas: 3
			image:    "myapp/api:v1.0.0"
		}
		database: {
			version:    "14.5"
			password:   "secure-db-password"
			volumeSize: "20Gi"
		}
		cache: {
			image:      "redis:7.0"
			volumeSize: "5Gi"
		}
		logAgent: {
			image: "prom/node-exporter:v1.6.1"
		}
		migration: {
			image:   "myapp/migrations:v2.0.0"
			version: "v2.0.0"
		}
		backup: {
			image:    "postgres:14.5"
			password: "secure-db-password"
			schedule: "0 2 * * *"
		}
	}
}

allBlueprintsModuleRelease: {
	metadata: {
		name:      "all-blueprints-release"
		namespace: "production"
	}
	#module: allBlueprintsModule

	values: {
		web: {
			replicas: 4
			image:    "nginx:1.21.6"
		}
		api: {
			replicas: 5
			image:    "myapp/api:v1.0.0"
		}
		database: {
			version:    "14.5"
			password:   "secure-db-password"
			volumeSize: "20Gi"
		}
		cache: {
			image:      "redis:7.0"
			volumeSize: "5Gi"
		}
		logAgent: {
			image: "prom/node-exporter:v1.6.1"
		}
		migration: {
			image:   "myapp/migrations:v2.0.0"
			version: "v2.0.0"
		}
		backup: {
			image:    "postgres:14.5"
			password: "secure-db-password"
			schedule: "0 2 * * *"
		}
	}
}

package data

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
	storage_resources "opmodel.dev/resources/storage@v1"
	workload_traits "opmodel.dev/traits/workload@v1"
)

/////////////////////////////////////////////////////////////////
//// SimpleDatabase Blueprint Definition
/////////////////////////////////////////////////////////////////

#SimpleDatabaseBlueprint: prim.#Blueprint & {
	metadata: {
		modulePath:  "opmodel.dev/blueprints/data"
		version:     "v1"
		name:        "simple-database"
		description: "A simple database workload with persistent storage"
	}

	composedResources: [
		workload_resources.#ContainerResource,
		storage_resources.#VolumesResource,
	]

	composedTraits: [
		workload_traits.#ScalingTrait,
		workload_traits.#RestartPolicyTrait,
	]

	spec: simpleDatabase: schemas.#SimpleDatabaseSchema
}

#SimpleDatabase: component.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "stateful"
	}

	#blueprints: (#SimpleDatabaseBlueprint.metadata.fqn): #SimpleDatabaseBlueprint

	workload_resources.#Container
	storage_resources.#Volumes
	workload_traits.#Scaling
	workload_traits.#RestartPolicy

	// Default/generated values - what WILL be generated
	spec: {
		simpleDatabase: schemas.#SimpleDatabaseSchema

		// Configure container based on database engine
		container: {
			name: "database"
			if simpleDatabase.engine == "postgres" {
				image: "postgres:\(simpleDatabase.version)"
			}
			if simpleDatabase.engine == "mysql" {
				image: "mysql:\(simpleDatabase.version)"
			}
			if simpleDatabase.engine == "mongodb" {
				image: "mongo:\(simpleDatabase.version)"
			}
			if simpleDatabase.engine == "redis" {
				image: "redis:\(simpleDatabase.version)"
			}
			env: {
				if simpleDatabase.engine == "postgres" {
					POSTGRES_DB: {
						name:  "POSTGRES_DB"
						value: simpleDatabase.dbName
					}
					POSTGRES_USER: {
						name:  "POSTGRES_USER"
						value: simpleDatabase.username
					}
					POSTGRES_PASSWORD: {
						name:  "POSTGRES_PASSWORD"
						value: simpleDatabase.password
					}
				}
				if simpleDatabase.engine == "mysql" {
					MYSQL_DATABASE: {
						name:  "MYSQL_DATABASE"
						value: simpleDatabase.dbName
					}
					MYSQL_USER: {
						name:  "MYSQL_USER"
						value: simpleDatabase.username
					}
					MYSQL_PASSWORD: {
						name:  "MYSQL_PASSWORD"
						value: simpleDatabase.password
					}
				}
				if simpleDatabase.engine == "mongodb" {
					MONGO_INITDB_DATABASE: {
						name:  "MONGO_INITDB_DATABASE"
						value: simpleDatabase.dbName
					}
					MONGO_INITDB_ROOT_USERNAME: {
						name:  "MONGO_INITDB_ROOT_USERNAME"
						value: simpleDatabase.username
					}
					MONGO_INITDB_ROOT_PASSWORD: {
						name:  "MONGO_INITDB_ROOT_PASSWORD"
						value: simpleDatabase.password
					}
				}
			}
			volumeMounts: {
				if simpleDatabase.persistence != _|_ && simpleDatabase.persistence.enabled {
					data: _dataMount
				}
			}

			if simpleDatabase.engine == "postgres" {
				readinessProbe: {
					exec: command: ["pg_isready", "-U", simpleDatabase.username]
					initialDelaySeconds: 5
					periodSeconds:       10
				}
			}
			if simpleDatabase.engine == "mysql" {
				readinessProbe: {
					exec: command: ["mysqladmin", "ping", "-h", "localhost"]
					initialDelaySeconds: 5
					periodSeconds:       10
				}
			}
			if simpleDatabase.engine == "mongodb" {
				readinessProbe: {
					exec: command: ["mongo", "--eval", "db.adminCommand('ping')"]
					initialDelaySeconds: 5
					periodSeconds:       10
				}
			}
			if simpleDatabase.engine == "redis" {
				readinessProbe: {
					exec: command: ["redis-cli", "ping"]
					initialDelaySeconds: 5
					periodSeconds:       10
				}
			}
		}

		// Helper for volume mount - defines where to mount in container
		_dataMount: {
			name: "data"
			if simpleDatabase.engine == "postgres" {
				mountPath: "/var/lib/postgresql/data"
			}
			if simpleDatabase.engine == "mysql" {
				mountPath: "/var/lib/mysql"
			}
			if simpleDatabase.engine == "mongodb" {
				mountPath: "/data/db"
			}
			if simpleDatabase.engine == "redis" {
				mountPath: "/data"
			}
		}

		// Helper for volume - defines storage source
		_dataVolume: {
			name: "data"
			persistentClaim: {
				size:       simpleDatabase.persistence.size
				accessMode: "ReadWriteOnce"
				if simpleDatabase.persistence.storageClass != _|_ {
					storageClass: simpleDatabase.persistence.storageClass
				}
			}
		}

		// Configure volumes if persistence is enabled
		if simpleDatabase.persistence != _|_ && simpleDatabase.persistence.enabled {
			volumes: {
				data: _dataVolume
			}
		}

		// Set scaling count to 1 (databases typically run single instance)
		scaling: count: 1

		// Always restart
		restartPolicy: "Always"
	}
}

// Test module for --skip-unprovided. Its "db" component attaches the
// provider-fulfilled backup trait of opmodel.dev/catalogs/opm@v4 while the
// backup value is true, its default. The catalog ships no provider for that
// trait, so against the module's own deps (or any platform without a backup
// provider) a default render refuses, and a render with --skip-unprovided
// renders both components and reports the skipped trait. With backup false
// the trait is not attached and nothing is skipped.
package backup_demo

import (
	m "opmodel.dev/core@v2"
	bp "opmodel.dev/catalogs/opm/blueprints/v1beta1"
	tra "opmodel.dev/catalogs/opm/traits/v1alpha1"

	id "example.com/modules/backup_demo/identity"
)

m.#Module

metadata: {
	name:       "backup_demo"
	modulePath: id.ModulePath
	version:    id.Version
}

#config: {
	// Attach the backup trait to the db component.
	backup: bool | *true
}

debugValues: {}

#components: {
	web: {
		bp.#StatelessWorkload

		metadata: name: "web"

		spec: statelessWorkload: {
			container: {
				name: "web"
				image: {
					repository: "nginx"
					tag:        "1.29"
					digest:     ""
				}
			}
			scaling: count: 1
			restartPolicy: "Always"
			updateStrategy: {
				type: "RollingUpdate"
				rollingUpdate: {}
			}
		}
	}

	db: {
		bp.#StatefulWorkload
		if #config.backup {
			tra.#Backup
		}

		metadata: name: "db"

		spec: {
			statefulWorkload: {
				volumes: data: {
					name: "data"
					persistentClaim: {
						size:         "1Gi"
						accessMode:   "ReadWriteOnce"
						storageClass: "standard"
					}
					readOnly: false
				}
				container: {
					name: "db"
					image: {
						repository: "valkey/valkey"
						tag:        "8"
						digest:     ""
					}
					volumeMounts: data: statefulWorkload.volumes.data & {
						mountPath: "/data"
					}
				}
				scaling: count: 1
				restartPolicy: "Always"
				updateStrategy: type: "RollingUpdate"
			}
			if #config.backup {
				backup: {
					schedule: "0 2 * * *"
					retention: keepDaily: 7
				}
			}
		}
	}
}

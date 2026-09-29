// Test module for --skip-unprovided. Its "db" component attaches the
// provider-fulfilled backup trait of opmodel.dev/catalogs/opm@v4 while the
// backup value is true, its default. The catalog ships no provider for that
// trait, so against the module's own deps (or any platform without a backup
// provider) a default render refuses, and a render with --skip-unprovided
// renders both components and reports the skipped trait. With backup false
// the trait is not attached and nothing is skipped. With stray true the db
// component also attaches #StrayTrait, a catalog-fulfilled contract nothing
// on any platform lists or implements: a gap --skip-unprovided never skips.
package backup_demo

import (
	m "opmodel.dev/core@v2"
	bp "opmodel.dev/catalogs/opm/blueprints/v1beta1"
	res "opmodel.dev/catalogs/opm/resources/v1beta1"
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
	// Attach the catalog-fulfilled stray trait to the db component.
	stray: bool | *false
}

// A load-bearing trait authored inline under a path outside every catalog,
// with the default catalog fulfilment: no enabled catalog lists or
// implements it, so a demand for it refuses whatever the switch.
#StrayTrait: m.#Trait & {
	metadata: {
		name:           "stray"
		modulePath:     "example.com/elsewhere/traits/v1"
		apiVersion:     "v1"
		catalogVersion: "0.0.1"
		fqn:            "example.com/elsewhere/traits/stray@v1"
		description:    "A catalog-fulfilled contract no catalog lists or implements"
	}
	optional: false
	appliesTo: [res.#VolumesResource]
	spec: stray: note!: string
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
		if #config.stray {
			#traits: (#StrayTrait.metadata.fqn): #StrayTrait
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
			if #config.stray {
				stray: note: "nobody lists me"
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

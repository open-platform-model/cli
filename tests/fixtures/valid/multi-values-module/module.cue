// Package main defines an OPM module used by the values-flow integration tests.
// It mirrors the real_module experiment fixture structure (same component/traits).
// values_prod.cue sits alongside this module to prove the loader filters it silently.
package main

import (
	"opmodel.dev/core@v1"
)

core.#Module

// Module metadata (v1alpha1 format — lightweight inline, no catalog import)
metadata: {
	modulePath:       "example.com/modules"
	name:             "multi-values-module"
	version:          "0.1.0"
	defaultNamespace: "default"
}

#config: {
	image: {
		repository: string | *"nginx"
		tag:        string | *"default"
		digest:     string | *""
	}
	replicas: int & >=1 | *1
}

#components: {
	web: {
		metadata: {
			name: "web"
			labels: {
				"core.opmodel.dev/workload-type": "stateless"
			}
		}

		#resources: {
			"opmodel.dev/resources/workload/container@v1": {
				metadata: {
					modulePath:  "opmodel.dev/resources/workload"
					version:     "v1"
					name:        "container"
					description: "Web container"
				}
				spec: container: {
					name!:  string
					image!: _
				}
			}
		}

		#traits: {
			"opmodel.dev/traits/workload/scaling@v1": {
				metadata: {
					modulePath:  "opmodel.dev/traits/workload"
					version:     "v1"
					name:        "scaling"
					description: "Scaling trait"
				}
				spec: scaling: {
					count: int & >=1 | *1
				}
			}
		}

		spec: {
			container: {
				name:  "web"
				image: #config.image
			}
			scaling: count: #config.replicas
		}
	}
}

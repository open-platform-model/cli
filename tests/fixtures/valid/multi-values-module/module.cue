// Package main defines an OPM module used by the values-flow integration tests.
// It mirrors the real_module experiment fixture structure (same component/traits)
// with the addition of values_prod.cue alongside values.cue — proving that
// extra values*.cue files in the module directory are harmless.
package main

import (
	"opmodel.dev/core@v0"
)

core.#Module

metadata: {
	apiVersion:       "example.com/multi-values-module@v0"
	name:             "multi-values-module"
	version:          "0.1.0"
	defaultNamespace: "default"
}

#config: {
	image:    string
	replicas: int & >=1 | *1
}

values: #config

#components: {
	web: core.#Component & {
		metadata: name: "web"

		#resources: {
			"opmodel.dev/resources/workload@v0#Container": core.#Resource & {
				metadata: {
					apiVersion:  "opmodel.dev/resources/workload@v0"
					name:        "container"
					description: "Web container"
				}
				#spec: container: {
					name!:  core.#NameType
					image!: string
				}
			}
		}

		#traits: {
			"opmodel.dev/traits/workload@v0#Scaling": core.#Trait & {
				metadata: {
					apiVersion:  "opmodel.dev/traits/workload@v0"
					name:        "scaling"
					description: "Scaling trait"
				}
				#spec: scaling: {
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

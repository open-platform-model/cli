// Package main defines the components for the simple experiment module.
//
// The "app" component is intentionally minimal but exercises the critical
// cross-reference pattern: spec fields reference #config.image and
// #config.replicas. These references are what we probe in the experiment —
// do they survive after module value reconstruction?
package main

import (
	"opmodel.dev/core@v0"
)

#components: {
	app: core.#Component & {
		metadata: name: "app"

		#resources: {
			"opmodel.dev/resources/workload@v0#Container": close(core.#Resource & {
				metadata: {
					apiVersion:  "opmodel.dev/resources/workload@v0"
					name:        "container"
					description: "App container resource"
					labels: "core.opmodel.dev/category": "workload"
				}
				// #spec fields become available in the component spec via _allFields.
				#spec: container: {
					name!:  core.#NameType
					image!: string
				}
			})
		}

		#traits: {
			"opmodel.dev/traits/workload@v0#Scaling": close(core.#Trait & {
				metadata: {
					apiVersion:  "opmodel.dev/traits/workload@v0"
					name:        "scaling"
					description: "Replica count scaling trait"
				}
				#spec: scaling: {
					count: int & >=1 | *1
				}
			})
		}

		// spec references #config — these are the cross-references under test.
		// After module reconstruction, these must still resolve when values are injected.
		spec: {
			container: {
				name:  "app"
				image: #config.image    // cross-reference: resolves via _#module: #module & {#config: values}
			}
			scaling: count: #config.replicas // cross-reference
		}
	}
}

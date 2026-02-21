// Package main defines a minimal OPM module that imports opmodel.dev/core@v0.
// Used by Strategy B (overlay) tests that prove Approach C:
//   - The module already imports opmodel.dev/core@v0
//   - The core package can be loaded from the module's resolved deps
//   - The loaded module cue.Value can be injected into #ModuleRelease.#module
//
// Requires OPM_REGISTRY to be set for dependency resolution.
package main

import (
	"opmodel.dev/core@v0"
)

// Apply the #Module constraint so this file IS a #Module when evaluated.
core.#Module

metadata: {
	apiVersion:       "example.com/exp-release-module@v0"
	name:             "exp-release-module"
	version:          "0.1.0"
	defaultNamespace: "default"
}

#config: {
	image:    string
	replicas: int & >=1 | *1
}

// values holds the module author's defaults. For the experiment fixture we keep
// these at schema-level (open to override) so that release values can unify
// without conflict. In production, module authors set concrete defaults here.
values: #config

// #components must use #resources/#traits so spec is derived via close({_allFields}).
// Direct spec.image would be rejected by #Component.spec: close({...}).
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

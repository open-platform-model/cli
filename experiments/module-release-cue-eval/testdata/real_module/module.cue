// Package main defines a minimal OPM module in v1alpha1 format.
// Used by builder tests that prove the BUILD phase pipeline:
//   - The module imports opmodel.dev/core@v1 (sub-package of opmodel.dev@v1)
//   - The core package is loaded from the module's resolved deps
//   - The loaded module cue.Value is injected into #ModuleRelease.#module
//
// Requires OPM_REGISTRY to be set for dependency resolution.
package main

import (
	"opmodel.dev/core@v1"
	schemas "opmodel.dev/schemas@v1"
	resources_workload "opmodel.dev/resources/workload@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
)

// Apply the #Module constraint so this file IS a #Module when evaluated.
core.#Module

metadata: {
	modulePath:       "example.com/modules"
	name:             "exp-release-module"
	version:          "0.1.0"
	defaultNamespace: "default"
}

#config: {
	image:    schemas.#Image
	replicas: int & >=1 | *1
}

// #components uses direct type composition from resources and traits packages.
#components: {
	web: {
		resources_workload.#Container
		traits_workload.#Scaling

		metadata: {
			name: "web"
			labels: "core.opmodel.dev/workload-type": "stateless"
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

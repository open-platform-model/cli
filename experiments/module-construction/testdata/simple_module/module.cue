// Package main defines a minimal OPM module for the module-construction experiment.
//
// Key design decisions reflected here:
//   - #config contains constraints only (no defaults)
//   - values is left as _ (inherited from core.#Module schema — unconstrained)
//   - No "values: #config" binding — that constraint belongs in #ModuleRelease
//   - Validation of values against #config is done in Go code (builder step 4)
//
// The components.cue file defines #components with cross-references to #config
// (e.g. #config.image, #config.replicas). These cross-references are the core
// subject of the experiment: do they survive module reconstruction?
package main

import (
	"opmodel.dev/core@v0"
)

// Apply the #Module constraint.
core.#Module

metadata: {
	apiVersion:       "test.dev/simple-module@v0"
	name:             "simple"
	version:          "0.1.0"
	defaultNamespace: "default"
}

// #config defines the value schema — constraints only, no defaults.
// Defaults live in values.cue (loaded separately via Pattern A).
#config: {
	image:    string
	replicas: int & >=1
	port:     int & >0
}

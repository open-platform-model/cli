// Package main provides concrete default values for the simple module.
//
// This file is loaded separately (Pattern A) — it is never unified into the
// module definition. LoadModule filters it out of load.Instances so that
// #config cross-references in #components remain abstract until build time.
//
// At build time, these values (or user-provided overrides) are injected into
// #ModuleRelease via FillPath("values", selectedValues), which causes CUE to
// evaluate _#module: #module & {#config: values} and resolve components.
package main

values: {
	image:    "nginx:latest"
	replicas: 1
	port:     8080
}

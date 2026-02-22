// values.cue — canonical default values for multi-values-module.
// Loaded separately by the loader (Pattern A) as mod.Values.
// These are the defaults used when no --values flag is provided.
package main

values: {
	image:    "nginx:default"
	replicas: 1
}

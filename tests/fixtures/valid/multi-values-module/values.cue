// values.cue — default values for multi-values-module.
// These match the #config defaults and serve as the fallback when no --values flag is passed.
package main

values: {
	image: {
		repository: "nginx"
		tag:        "default"
		digest:     ""
	}
	replicas: 1
}

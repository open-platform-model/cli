// Author defaults — the sane starting point defined by the module author.
// Loaded in isolation (not as part of load.Instances) to avoid CUE
// unification conflicts when multiple values files are present.
package main

values: {
	image:    "app:latest"
	replicas: 2
	port:     8080
	debug:    false
	env:      "dev"
}

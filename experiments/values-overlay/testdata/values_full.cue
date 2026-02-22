// Complete user values — all five fields present, standalone.
// An approach that produces a fully concrete result from this file alone
// does not need author defaults to fill any gaps.
package main

values: {
	image:    "app:2.0.0"
	replicas: 3
	port:     9090
	debug:    false
	env:      "prod"
}

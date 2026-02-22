// Minimal module fixture for the values-overlay experiment.
//
// The module defines a five-field #config schema covering the common
// value types: string, int (bounded), bool, and a string enum.
// values.cue (loaded separately) provides the author's default values.
package main

metadata: {
	name:    "app"
	version: "1.0.0"
}

// #config is the value schema — constraints only, no concrete defaults.
#config: {
	// Container image to deploy.
	image: string

	// Number of replicas to run (1–10).
	replicas: int & >=1 & <=10

	// TCP port the application listens on.
	port: int & >0 & <=65535

	// Enable debug logging.
	debug: bool

	// Target environment — controls logging verbosity and feature flags.
	env: "dev" | "staging" | "prod"
}

// values must satisfy #config.
// Concrete defaults live in values.cue, loaded separately to avoid
// package-level unification conflicts (see values-load-isolation experiment).
values: #config

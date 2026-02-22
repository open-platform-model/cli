// Minimal test module that mirrors the minecraft pattern:
// - #config schema with an enum field (serverType)
// - values: #config  (so values.cue must satisfy the schema)
// - Multiple values_*.cue files in the same package cause CUE unification conflicts
package main

// Module metadata
metadata: {
	name:             "test-server"
	version:          "1.0.0"
	fqn:              "example.com/test-server@v0"
	defaultNamespace: "default"
}

// #config is the schema. Fields are constraints — no concrete defaults here.
#config: {
	// Server software type — the field that will conflict across values files
	serverType: "VANILLA" | "PAPER" | "FORGE" | "FABRIC" | "SPIGOT"

	// Port the server listens on
	port: int & >0 & <=65535

	// Maximum concurrent players
	maxPlayers: int & >0 & <=1000
}

// values must satisfy #config — concrete values live in values.cue
values: #config


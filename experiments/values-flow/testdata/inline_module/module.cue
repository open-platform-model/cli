// inline_module/module.cue — pattern B: inline concrete values, multi-file package.
//
// The values field holds concrete defaults directly in module.cue.
// There is no values.cue — Approach A has nothing to filter.
// The package is split across module.cue + components.cue to prove that
// Approach A loads ALL non-values files correctly even when no filtering occurs.
package inlinemodule

apiVersion: "opmodel.dev/core/v0"
kind:       "Module"

metadata: {
	apiVersion: "example.com/inline-module@v0"
	name:       "inline-module"
	version:    "1.0.0"
}

// Schema — constraints only.
#config: {
	image:    string
	replicas: int & >=1
}

// Inline concrete defaults — the "inline values" author pattern.
// Distinct from values_module defaults ("nginx:latest") to make assertions unambiguous.
values: {
	image:    "nginx:stable"
	replicas: 2
}

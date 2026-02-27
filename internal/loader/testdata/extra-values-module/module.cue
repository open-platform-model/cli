package extravaluesmodule

// extra-values-module: fixture for testing that extra values*.cue files in the
// module directory are silently filtered out and do not affect the loaded defaults.
// values.cue provides the canonical defaults; values_prod.cue is present but ignored
// by the loader (it can be passed via --values at build time).

metadata: {
	modulePath: "example.com/modules"
	name:       "extra-values-module"
	version:    "1.0.0"
	fqn:        "example.com/modules/extra-values-module:1.0.0"
}

#config: {
	image: {
		repository: string
		tag:        string
		digest:     string
	}
	replicas: int & >=1
}

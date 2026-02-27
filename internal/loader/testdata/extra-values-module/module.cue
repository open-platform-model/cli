package extravaluesmodule

// extra-values-module: fixture for testing that extra values*.cue files in the
// module directory are silently filtered out by the loader.
// values_prod.cue is present but ignored by the loader; pass it via --values at build time.
// Defaults are defined in #config — no values.cue is needed.

metadata: {
	modulePath: "example.com/modules"
	name:       "extra-values-module"
	version:    "1.0.0"
	fqn:        "example.com/modules/extra-values-module:1.0.0"
}

#config: {
	image: {
		repository: string | *"nginx"
		tag:        string | *"default"
		digest:     string | *""
	}
	replicas: int & >=1 | *1
}

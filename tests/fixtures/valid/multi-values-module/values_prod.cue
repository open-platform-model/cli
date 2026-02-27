// values_prod.cue — production environment overrides.
// This file lives in the module directory intentionally to prove that the loader
// filters it out silently. Pass via --values at build time to apply these values.
package main

values: {
	image: {
		repository: "nginx"
		tag:        "prod"
		digest:     ""
	}
	replicas: 3
}

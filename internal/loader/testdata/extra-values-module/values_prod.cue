package extravaluesmodule

// Production overrides — filtered out by the loader during package load.
// Pass via --values at build time to apply these values.
values: {
	image: {
		repository: "nginx"
		tag:        "prod"
		digest:     ""
	}
	replicas: 3
}

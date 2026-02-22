package extravaluesmodule

// Production overrides — filtered out by the loader during package load.
// Pass via --values at build time to apply these values.
values: {
	image:    "nginx:prod"
	replicas: 3
}

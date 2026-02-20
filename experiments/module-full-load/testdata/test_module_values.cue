package values

// values overrides are split into a separate file — matching the real module pattern.
// The build phase loads this file and unifies it with the base value.
values: {
	image:    "nginx:1.28.2"
	replicas: 3
}

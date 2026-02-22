// Partial user values — only two of the five fields are specified.
// image and replicas are overridden; port, debug, and env are intentionally
// absent. Approaches that implement defaults-fallthrough must fill the gaps
// from the author defaults (values.cue).
package main

values: {
	image:    "app:1.2.3"
	replicas: 5
}

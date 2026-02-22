// values.cue — module author defaults for the real_module fixture.
// Loaded separately via Approach A (ctx.CompileBytes) so it does not cause
// a unification conflict when multiple values*.cue files are present.
package main

values: {
	image:    "nginx:1.0"
	replicas: 1
}

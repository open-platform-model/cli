// values.cue — module author defaults for the fake_module fixture.
// Loaded separately via Approach A (ctx.CompileBytes) so it does not cause
// a unification conflict when multiple values*.cue files are present.
package expmodule

values: {
	image:    "nginx:latest"
	replicas: 1
}

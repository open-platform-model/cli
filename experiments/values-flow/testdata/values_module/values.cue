// values_module/values.cue — module author defaults (Layer 1).
// Loaded separately via Approach A (ctx.CompileBytes), never via load.Instances.
// These values are distinct from user_values.cue to make test assertions unambiguous.
package valuesmodule

values: {
	image:    "nginx:latest"
	replicas: 1
}

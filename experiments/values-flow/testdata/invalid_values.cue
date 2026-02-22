// invalid_values.cue — values that violate the #config schema.
//
// replicas: 0 violates the constraint `int & >=1` defined in both
// values_module and inline_module's #config.
// Used to prove schema validation fires correctly after selectValues().
// No package declaration — values files may be package-free.
values: {
	image:    "nginx:1.0"
	replicas: 0
}

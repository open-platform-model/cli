package testmodule

// Intentionally incomplete: missing 'replicas', leaving #config.replicas
// as "int & >=1" (non-concrete) in components after FillPath.
values: {
	image: "nginx:1.27"
	port:  9090
}

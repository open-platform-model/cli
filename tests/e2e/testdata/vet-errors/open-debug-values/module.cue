package demo

// A values-validation fixture, not a full core #Module: the kernel loader
// gate publish and vet share only needs kind plus concrete identity.
//
// #config requires a field with no default and debugValues is left open, so
// the merged value is not concrete: vet refuses it as a #config violation
// reported at the schema's own position, exactly as build does.
kind: "Module"
metadata: {
	name:       "demo-open"
	modulePath: "test.example.com/demo-open@v0"
	version:    "0.1.0"
}

#config: {
	replicas: int
}

debugValues: _

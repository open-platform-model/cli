package demo

// A values-validation fixture, not a full core #Module: the kernel loader
// gate publish and vet share only needs kind plus concrete identity.
//
// No #config at all: values files have nothing to be checked against, so
// vet refuses them exactly as build does.
kind: "Module"
metadata: {
	name:       "demo-noconfig"
	modulePath: "test.example.com/demo-noconfig@v0"
	version:    "0.1.0"
}

debugValues: {}

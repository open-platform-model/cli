package demo

// A values-validation fixture, not a full core #Module: the kernel loader
// gate publish and vet share only needs kind plus concrete identity.
kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "test.example.com/demo@v0"
	version:    "0.1.0"
}

#config: {
	media?: [Name=string]: {
		type: "pvc" | *"emptyDir"
	}
}

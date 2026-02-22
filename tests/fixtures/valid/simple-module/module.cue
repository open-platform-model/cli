package simplemodule

// Module metadata
metadata: {
	apiVersion: "example.com/modules@v0"
	name:       "simple-module"
	version:    "0.1.0"
}

// Abstract value schema with CUE defaults — required for manifests to evaluate.
// Concrete defaults live in values.cue (Pattern A); this provides the type
// context so that references to values.* in manifests resolve during package load.
values: {
	replicas: *1 | int
	image:    *"nginx:latest" | string
}

// Output manifests
manifests: [
	{
		apiVersion: "apps/v1"
		kind:       "Deployment"
		metadata: name: metadata.name
		spec: {
			replicas: values.replicas
			selector: matchLabels: app: metadata.name
			template: {
				metadata: labels: app: metadata.name
				spec: containers: [{
					name:  metadata.name
					image: values.image
				}]
			}
		}
	},
]

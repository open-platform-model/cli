package simplemodule

// Module metadata (v1alpha1 format)
metadata: {
	modulePath: "example.com/modules"
	name:       "simple-module"
	version:    "0.1.0"
	fqn:        "example.com/modules/simple-module:0.1.0"
}

// Configuration schema with defaults.
// In v1alpha1, defaults live in #config — no separate values.cue is needed.
#config: {
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
			replicas: #config.replicas
			selector: matchLabels: app: metadata.name
			template: {
				metadata: labels: app: metadata.name
				spec: containers: [{
					name:  metadata.name
					image: #config.image
				}]
			}
		}
	},
]

package main

// Simple Kubernetes provider with one transformer
provider: #Provider & {
	metadata: {
		name:        "test-kubernetes"
		version:     "0.1.0"
		minVersion:  "0.1.0"
		description: "Test Kubernetes provider"
	}

	transformers: {
		"deployment": #Transformer & {
			metadata: {
				name:        "DeploymentTransformer"
				apiVersion:  "transformer.opm.dev/workload@v0"
				description: "Transforms stateless workloads into Kubernetes Deployments"
			}
			requiredLabels: {
				"core.opm.dev/workload-type": "stateless"
			}
			requiredResources: {}
			optionalResources: {}
			requiredTraits:    {}
			optionalTraits:    {}

			#transform: {
				#component: _
				context:    #TransformerContext

				output: {
					apiVersion: "apps/v1"
					kind:       "Deployment"
					metadata: {
						name:      #component.metadata.name
						namespace: context.namespace
						labels:    context.labels
					}
					spec: {
						replicas: 1
						selector: matchLabels: "app": #component.metadata.name
						template: {
							metadata: labels: "app": #component.metadata.name
							spec: containers: [{
								name:  #component.metadata.name
								image: "nginx:latest"
							}]
						}
					}
				}
			}
		}
	}
}

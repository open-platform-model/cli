package main

import (
	core "opm.dev/core@v0"
	modules "opm.dev/examples/modules@v0"
)

// 1. Instantiate the release from the imported module
// Fill in required release metadata (name, namespace)
release: modules.basicModuleRelease



// 2. Define transformers that match the spec contract (013-cli-render-spec)
// Each transformer declares matching criteria and a #transform function
// that takes #component + context and produces a single Kubernetes resource.

_exampleContext: core.#TransformerContext & {
    #moduleMetadata: release.metadata
    #componentMetadata: release.components.web.metadata

    name:      "basic-app"
    namespace: "production"

    labels: {
        "environment": "production"
    }
}

#DeploymentTransformer: {
	// Metadata (per spec contract)
	metadata: {
		name:        "DeploymentTransformer"
		description: "Converts stateless workload components to Kubernetes Deployments"
		version:     "v1"
	}

	// Matching criteria
	requiredResources: {
		"opm.dev/resources/workload@v0#Container": _
	}
	requiredLabels: {
		"core.opm.dev/workload-type": "stateless"
	}

	// Transform function - converts OPM component to K8s Deployment
	#transform: {
		#component: _
		context:   core.#TransformerContext

		// Helper to convert OPM ports to K8s container ports
		let _ports = [
			for portName, port in #component.spec.container.ports {
				name:          port.name
				containerPort: port.targetPort
				protocol:      *port.protocol | "TCP"
			},
		]

		// Helper to convert OPM env to K8s env vars
		let _env = [
			for envName, env in #component.spec.container.env {
				name:  env.name
				value: env.value
			},
		]

		// Helper to convert OPM volumeMounts if present
		let _volumeMounts = [
			if #component.spec.container.volumeMounts != _|_
			for vmName, vm in #component.spec.container.volumeMounts {
				name:      vm.name
				mountPath: vm.mountPath
			},
		]

		// Helper to convert OPM volumes to K8s volumes
		let _volumes = [
			for volName, vol in #component.spec.volumes {
				name: vol.name
				if vol.persistentClaim != _|_ {
					persistentVolumeClaim: claimName: vol.name
				}
			},
		]

		output: {
			apiVersion: "apps/v1"
			kind:       "Deployment"
			metadata: {
				name:      #component.metadata.name
				namespace: context.namespace
				labels: context.labels
			}
			spec: {
				replicas: #component.spec.replicas
				selector: matchLabels: {
					"app.kubernetes.io/name": #component.metadata.name
				}
				template: {
					metadata: labels: {
						"app.kubernetes.io/name": #component.metadata.name
					}
					spec: {
						containers: [{
							name:            #component.spec.container.name
							image:           #component.spec.container.image
							imagePullPolicy: #component.spec.container.imagePullPolicy
							ports:           _ports
							env:             _env
							if #component.spec.container.resources != _|_ {
								resources: #component.spec.container.resources
							}
							if len(_volumeMounts) > 0 {
								volumeMounts: _volumeMounts
							}
						}]
						if len(_volumes) > 0 {
							volumes: _volumes
						}
					}
				}
			}
		}
	}
}

#StatefulSetTransformer: {
	// Metadata (per spec contract)
	metadata: {
		name:        "StatefulSetTransformer"
		description: "Converts stateful workload components to Kubernetes StatefulSets"
		version:     "v1"
	}

	// Matching criteria
	requiredResources: {
		"opm.dev/resources/workload@v0#Container": _
	}
	requiredLabels: {
		"core.opm.dev/workload-type": "stateful"
	}

	// Transform function - converts OPM component to K8s StatefulSet
	#transform: {
		#component: _
		context:   core.#TransformerContext

		// Helper to convert OPM ports to K8s container ports
		let _ports = [
			for portName, port in #component.spec.container.ports {
				name:          port.name
				containerPort: port.targetPort
				protocol:      *port.protocol | "TCP"
			},
		]

		// Helper to convert OPM env to K8s env vars
		let _env = [
			for envName, env in #component.spec.container.env {
				name:  env.name
				value: env.value
			},
		]

		// Helper to convert OPM volumeMounts
		let _volumeMounts = [
			if #component.spec.container.volumeMounts != _|_
			for vmName, vm in #component.spec.container.volumeMounts {
				name:      vm.name
				mountPath: vm.mountPath
			},
		]

		// Helper to convert OPM volumes to K8s volumeClaimTemplates
		let _volumeClaimTemplates = [
			for volName, vol in #component.spec.volumes if vol.persistentClaim != _|_ {
				metadata: name: vol.name
				spec: {
					accessModes: [*vol.persistentClaim.accessMode | "ReadWriteOnce"]
					resources: requests: storage: vol.persistentClaim.size
					if vol.persistentClaim.storageClass != _|_ {
						storageClassName: vol.persistentClaim.storageClass
					}
				}
			},
		]

		output: {
			apiVersion: "apps/v1"
			kind:       "StatefulSet"
			metadata: {
				name:      #component.metadata.name
				namespace: context.namespace
				labels: context.labels
			}
			spec: {
				serviceName: #component.metadata.name
				replicas:    #component.spec.replicas
				selector: matchLabels: {
					"app.kubernetes.io/name": #component.metadata.name
				}
				template: {
					metadata: labels: {
						"app.kubernetes.io/name": #component.metadata.name
					}
					spec: containers: [{
						name:            #component.spec.container.name
						image:           #component.spec.container.image
						imagePullPolicy: #component.spec.container.imagePullPolicy
						ports:           _ports
						env:             _env
						if #component.spec.container.resources != _|_ {
							resources: #component.spec.container.resources
						}
						if len(_volumeMounts) > 0 {
							volumeMounts: _volumeMounts
						}
					}]
				}
				if len(_volumeClaimTemplates) > 0 {
					volumeClaimTemplates: _volumeClaimTemplates
				}
			}
		}
	}
}

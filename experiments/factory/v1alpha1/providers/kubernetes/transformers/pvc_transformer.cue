package transformers

import (
	transformer "opmodel.dev/core/transformer@v1"
	storage_resources "opmodel.dev/resources/storage@v1"
	k8scorev1 "opmodel.dev/schemas/kubernetes/core/v1@v1"
)

// PVCTransformer creates standalone PersistentVolumeClaims from Volume resources
#PVCTransformer: transformer.#Transformer & {
	metadata: {
		modulePath:  "opmodel.dev/providers/kubernetes/transformers"
		version:     "v1"
		name:        "pvc-transformer"
		description: "Creates standalone Kubernetes PersistentVolumeClaims from Volume resources"

		labels: {
			"core.opmodel.dev/resource-category": "storage"
			"core.opmodel.dev/resource-type":     "persistentvolumeclaim"
		}
	}

	requiredLabels: {} // No specific labels required; matches any component with Volumes resource

	// Required resources - Volumes MUST be present
	requiredResources: {
		"opmodel.dev/resources/storage/volumes@v1": storage_resources.#VolumesResource
	}

	// No optional resources
	optionalResources: {}

	// No required traits
	requiredTraits: {}

	// No optional traits
	optionalTraits: {}

	#transform: {
		#component: _ // Unconstrained; validated by matching, not by transform signature
		#context:   transformer.#TransformerContext

		// Extract required Volumes resource (will be bottom if not present)
		_volumes: #component.spec.volumes

		// Generate PVC for each volume that has a persistentClaim defined
		output: {
			for volumeName, volume in _volumes if volume.persistentClaim != _|_ {
				"\(volumeName)": k8scorev1.#PersistentVolumeClaim & {
					apiVersion: "v1"
					kind:       "PersistentVolumeClaim"
					metadata: {
						name:      volume.name | *volumeName
						namespace: #context.#moduleReleaseMetadata.namespace
						labels:    #context.labels
						// Include component annotations if present
						if len(#context.componentAnnotations) > 0 {
							annotations: #context.componentAnnotations
						}
					}
					spec: {
						// accessMode is singular in schema, K8s expects accessModes array
						accessModes: [volume.persistentClaim.accessMode | *"ReadWriteOnce"]
						resources: {
							requests: {
								storage: volume.persistentClaim.size
							}
						}

						if volume.persistentClaim.storageClass != _|_ {
							storageClassName: volume.persistentClaim.storageClass
						}
					}
				}
			}
		}
	}
}

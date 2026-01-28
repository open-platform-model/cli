package core

import (
	"list"
)

#Provider: {
	apiVersion: "core.opm.dev/v0"
	kind:       "Provider"
	metadata: {
		name:        string // The name of the provider
		description: string // A brief description of the provider
		version:     string // The version of the provider
		minVersion:  string // The minimum version of the provider

		// Labels for provider categorization and compatibility
		// Example: {"core.opm.dev/format": "kubernetes"}
		labels?: #LabelsAnnotationsType
	}

	// Transformer registry - maps platform resources to transformers
	// Example:
	// transformers: {
	// 	"k8s.io/api/apps/v1.Deployment": #DeploymentTransformer
	// 	"k8s.io/api/apps/v1.StatefulSet": #StatefulSetTransformer
	// }
	transformers: #TransformerMap

	// All resources, traits declared by transformers
	// Extract FQNs from the map keys
	#declaredResources: list.FlattenN([
		for _, transformer in transformers {
			list.Concat([
				[for fqn, _ in transformer.requiredResources {fqn}],
				[for fqn, _ in transformer.optionalResources {fqn}],
			])
		},
	], 1)

	#declaredTraits: list.FlattenN([
		for _, transformer in transformers {
			list.Concat([
				[for fqn, _ in transformer.requiredTraits {fqn}],
				[for fqn, _ in transformer.optionalTraits {fqn}],
			])
		},
	], 1)

	#declaredDefinitions: list.Concat([#declaredResources, #declaredTraits])
	...
}

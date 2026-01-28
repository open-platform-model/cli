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

// #MatchTransformers computes the matching plan for a render pipeline.
// Maps each transformer to its list of matched components.
// Corresponds to Phase 3 (Component Matching) of the render pipeline (013-cli-render-spec).
//
// The output is a map where:
//   - Keys are transformer IDs (same as provider.transformers keys)
//   - Values contain the transformer definition and list of matching components
//   - Only transformers with at least one match are included
//
// Usage:
//   let plan = (#MatchTransformers & {provider: myProvider, module: myRelease}).out
//   for transformerID, match in plan {
//       // match.transformer: the transformer definition
//       // match.components: list of components that matched
//   }
#MatchTransformers: {
	provider: #Provider
	module:   #ModuleRelease

	out: {
		// Capture parameters as local aliases for use in comprehensions
		let _prov = provider
		let _mod = module
		
		// Iterate over all transformers in the provider
		for tID, t in _prov.transformers {
			// Find all components in the module that match this transformer
			let matches = [
				for _, c in _mod.components
				if (#Matches & {transformer: t, component: c}).result {
					c
				},
			]

			// Only include this transformer if it matched at least one component
			if len(matches) > 0 {
				(tID): {
					transformer: t
					components:  matches
				}
			}
		}
	}
}

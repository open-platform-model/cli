package experiment

import (
	core "test.com/experiment/pkg/core"
)

// #ProviderWithMatching extends the core Provider to include logic
// for matching components to transformers.
#ProviderWithMatching: core.#Provider & {
	// Re-declare transformers to ensure it's available in this scope
	transformers: _

	// Input: A complete OPM Module
	#module: core.#ModuleRelease

	// Output: A map where keys are transformer IDs and values contain
	// the transformer definition and the list of matching components.
	// This corresponds to Phase 3 of the Render Pipeline.
	#matchedTransformers: {
		// Use local aliases to capture scope for comprehension
		let _transformers = transformers
		let _comps = #module.components

		for tID, t in _transformers {
			// Find all components that match this transformer
			let matches = [
				for cID, c in _comps
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

	// Output: A flat map of rendered platform resources.
	// Keys are "<transformerID>/<componentName>", values are the transformer output.
	// This corresponds to Phase 4 of the Render Pipeline.
	#rendered: {
		let _release = #module

		for tID, match in #matchedTransformers {
			for _, comp in match.components {
				"\(tID)/\(comp.metadata.name)": (match.transformer.#transform & {
					#component: comp
					context: core.#TransformerContext & {
						#moduleMetadata:    _release.metadata
						#componentMetadata: comp.metadata
						name:      _release.metadata.name
						namespace: _release.metadata.namespace
					}
				}).output
			}
		}
	}
}

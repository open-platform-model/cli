package core

// #Transformer: Declares how to convert OPM components into platform-specific resources.
//
// Transformers use label-based matching to determine which components they can handle.
// A transformer matches a component when ALL of the following are true:
//   1. ALL requiredLabels are present on the component with matching values
//   2. ALL requiredResources FQNs exist in component.#resources
//   3. ALL requiredTraits FQNs exist in component.#traits
//
// Component labels are inherited from the union of labels from all attached
// #resources, #traits, and #policies definitions.
#Transformer: {
	apiVersion: "opm.dev/core/v0"
	kind:       "Transformer"

	metadata: {
		apiVersion!: #NameType                          // Example: "opm.dev/transformers/kubernetes@v0"
		name!:       #NameType                          // Example: "DeploymentTransformer"
		fqn:         #FQNType & "\(apiVersion)#\(name)" // Example: "opm.dev/transformers/kubernetes@v0#DeploymentTransformer"

		description!: string // A brief description of what this transformer produces

		// Labels for categorizing this transformer (not used for matching)
		labels?: #LabelsAnnotationsType

		// Annotations for additional transformer metadata
		annotations?: #LabelsAnnotationsType
	}

	// Labels that a component MUST have to match this transformer.
	// Component labels are inherited from the union of labels from all attached
	// #resources, #traits, and #policies.
	//
	// Example: A DeploymentTransformer requires stateless workloads:
	//   requiredLabels: {"core.opm.dev/workload-type": "stateless"}
	//
	// The Container resource defines this label, so components with Container
	// will have it. Transformers requiring "stateful" won't match.
	requiredLabels?: #LabelsAnnotationsType

	// Labels optionally used by this transformer - component MAY include these
	// If not provided, defaults from the definition can be used
	optionalLabels?: #LabelsAnnotationsType

	// Resources required by this transformer - component MUST include these
	// Map key is the FQN, value is the Resource definition (provides access to #defaults)
	requiredResources: [string]: _

	// Resources optionally used by this transformer - component MAY include these
	// If not provided, defaults from the definition can be used
	optionalResources: [string]: _

	// Traits required by this transformer - component MUST include these
	// Map key is the FQN, value is the Trait definition (provides access to #defaults)
	requiredTraits: [string]: _

	// Traits optionally used by this transformer - component MAY include these
	// If not provided, defaults from the definition can be used
	optionalTraits: [string]: _

	// Transform function
	// IMPORTANT: output must be a single resource
	#transform: {
		#component: _ // Unconstrained; validated by matching, not by the transform signature
		context:   #TransformerContext

		output: {...} // Must be a single provider-specific resource
	}
}

// Map of transformers by fully qualified name
#TransformerMap: [string]: #Transformer

// Provider context passed to transformers
#TransformerContext: close({
	#moduleMetadata: _ // Injected during rendering
	#componentMetadata: _ // Injected during rendering
	name:      string // Injected during rendering
	namespace: string // Injected during rendering

	moduleLabels: {
		if #moduleMetadata.labels != _|_ {#moduleMetadata.labels}
	}

	componentLabels: {
		"app.kubernetes.io/instance":  "\(name)-\(namespace)"

		if #componentMetadata.labels != _|_ {#componentMetadata.labels}
	}

	controllerLabels: {
		"app.kubernetes.io/managed-by": "open-platform-model"
		"app.kubernetes.io/name":       #componentMetadata.name
		"app.kubernetes.io/version":    #moduleMetadata.version
	}

	labels: {[string]: string}
	labels: {
		for k, v in moduleLabels {
			(k): "\(v)"
		}
		for k, v in componentLabels {
			(k): "\(v)"
		}
		for k, v in controllerLabels {
			(k): "\(v)"
		}
		...
	}
})

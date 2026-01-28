package core

// #Bundle: Defines a collection of modules. Bundles enable grouping
// related modules for easier distribution and management.
// Bundles can contain multiple modules, each representing a set of
// definitions (resources, traits, blueprints, policies, scopes).
#Bundle: close({
	apiVersion: "opm.dev/core/v0"
	kind:       "Bundle"

	metadata: {
		apiVersion!: #NameType                          // Example: "opm.dev/bundles/core@v0"
		name!:       #NameType                          // Example: "ExampleBundle"
		fqn:         #FQNType & "\(apiVersion)#\(name)" // Example: "opm.dev/bundles/core@v0#ExampleBundle"

		// Human-readable description of the bundle
		description?: string

		// Optional metadata labels for categorization and filtering
		labels?: #LabelsAnnotationsType

		// Optional metadata annotations for bundle behavior hints
		annotations?: #LabelsAnnotationsType
	}

	// Modules included in this bundle (full references)
	#modules!: #ModuleMap

	// MUST be an OpenAPIv3 compatible schema

	// Value schema - constraints only, NO defaults
	// Developers define the configuration contract
	// Platform teams can add defaults and refine constraints via CUE merging
	// MUST be OpenAPIv3 compliant (no CUE templating - for/if statements)
	#spec!: _

	// Concrete values - should contain sane default values
	// Developers define these values but it can be overriden by the platform operator.
	// The end-user's concrete values override this except if a platform operator has already defined them.
	values: _
})

#BundleDefinitionMap: [string]: #Bundle

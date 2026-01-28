package core

#CompiledBundle: close({
	apiVersion: "opm.dev/core/v0"
	kind:       "Bundle"

	metadata: {
		apiVersion!: #NameType                          // Example: "opm.dev/bundles@v0"
		name!:       #NameType                          // Example: "ExampleBundle"
		fqn:         #FQNType & "\(apiVersion)#\(name)" // Example: "opm.dev/bundles@v0#ExampleBundle"

		// Human-readable description of the bundle
		description?: string

		// Optional metadata labels for categorization and filtering
		labels?: #LabelsAnnotationsType

		// Optional metadata annotations for bundle behavior hints
		annotations?: #LabelsAnnotationsType
	}

	// Modules included in this bundle (full references)
	#modules!: #CompiledModuleMap

	// MUST be an OpenAPIv3 compatible schema
	#values!: _

	// Concerete values (preserved from Module)
	values: _
})

#CompiledBundleMap: [string]: #CompiledBundle

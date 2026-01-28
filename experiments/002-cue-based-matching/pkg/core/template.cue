package core

/////////////////////////////////////////////////////////////////
//// Template Definition
/////////////////////////////////////////////////////////////////

// #TemplateDefinition: Defines a module template that can be used to
// initialize new OPM modules. Templates provide starting points for
// different use cases and complexity levels.
#TemplateDefinition: close({
	apiVersion: #NameType & "opm.dev/core/v0"
	kind:       #NameType & "Template"

	metadata: {
		apiVersion!: #NameType                          // Example: "templates.opm.dev/core@v1"
		name!:       #NameType                          // Example: "Standard"
		fqn:         #FQNType & "\(apiVersion)#\(name)" // Example: "templates.opm.dev/core@v1#Standard"

		// Template category: "module" or "bundle"
		category!: "module" | "bundle"

		// Human-readable description of the template
		description?: string

		// Template complexity level (e.g., "beginner", "intermediate", "advanced")
		level?: "beginner" | "intermediate" | "advanced"

		// Primary use case for this template
		useCase?: string

		// Optional metadata labels for categorization and filtering
		labels?: #LabelsAnnotationsType

		// Optional metadata annotations for additional information
		annotations?: #LabelsAnnotationsType
	}
})

#TemplateMap: [string]: #TemplateDefinition

#TemplateStringArray: [...string]

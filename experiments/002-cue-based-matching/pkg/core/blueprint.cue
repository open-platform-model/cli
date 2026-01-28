package core

import (
	"strings"
)

// #Blueprint: Defines a reusable blueprint
// that composes resources and traits into a higher-level abstraction.
// Blueprints enable standardized configurations for common use cases.
#Blueprint: close({
	apiVersion: "opm.dev/core/v0"
	kind:       "Blueprint"

	metadata: {
		apiVersion!: #NameType                          // Example: "opm.dev/blueprints@v0"
		name!:       #NameType                          // Example: "StatelessWorkload"
		fqn:         #FQNType & "\(apiVersion)#\(name)" // Example: "opm.dev/blueprints@v0#StatelessWorkload"

		// Human-readable description of the definition
		description?: string

		// Optional metadata labels for categorization and filtering
		// Labels are used by OPM for definition selection and matching
		// Example: {"core.opm.dev/workload-type": "stateless"}
		labels?: #LabelsAnnotationsType

		// Optional metadata annotations for definition behavior hints (not used for categorization)
		// Annotations provide additional metadata but are not used for selection
		annotations?: #LabelsAnnotationsType
	}

	// Resources that compose this blueprint (full references)
	composedResources!: [...#Resource]

	// Traits that compose this blueprint (full references)
	composedTraits?: [...#Trait]

	// MUST be an OpenAPIv3 compatible schema
	// The field and schema exposed by this definition
	// Use # to allow inconcrete fields
	// TODO: Add OpenAPIv3 schema validation
	#spec!: (strings.ToCamel(metadata.name)): _
})

#BlueprintMap: [string]: #Blueprint

#BlueprintStringArray: [...string]

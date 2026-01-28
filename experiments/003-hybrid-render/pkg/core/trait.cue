package core

import (
	"strings"
)

// #Trait: Defines additional behavior or characteristics
// that can be attached to components.
#Trait: close({
	apiVersion: "opm.dev/core/v0"
	kind:       "Trait"

	metadata: {
		apiVersion!: #NameType                          // Example: "opm.dev/traits/scaling@v0"
		name!:       #NameType                          // Example: "Replicas"
		fqn:         #FQNType & "\(apiVersion)#\(name)" // Example: "opm.dev/traits/scaling@v0#Replicas"

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

	// MUST be an OpenAPIv3 compatible schema
	// The field and schema exposed by this definition
	// Use # to allow inconcrete fields
	// TODO: Add OpenAPIv3 schema validation
	#spec!: (strings.ToCamel(metadata.name)): _

	// Resources that this trait can be applied to (full references)
	appliesTo!: [...#Resource]
})

#TraitMap: [string]: _

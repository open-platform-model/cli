package core

import (
	"strings"
)

// #Resource: Defines a resource of deployment within the system.
// Resources represent deployable components, services or resources
// that can be instantiated and managed independently.
#Resource: close({
	apiVersion: #NameType & "opm.dev/core/v0"
	kind:       #NameType & "Resource"

	metadata: {
		apiVersion!: #NameType                          // Example: "resources.opm.dev/workload@v0"
		name!:       #NameType                          // Example: "Container"
		fqn:         #FQNType & "\(apiVersion)#\(name)" // Example: "resources.opm.dev/workload@v0#Container"

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
})

#ResourceMap: [string]: _

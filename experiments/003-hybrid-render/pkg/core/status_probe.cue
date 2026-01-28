package core

import (
	"strings"
)

// #StatusProbe: Defines a reusable runtime health check query.
// Probes are evaluated by the platform controller against the live system state.
// They utilize native CUE logic to determine health based on inputs.
#StatusProbe: close({
	apiVersion: "opm.dev/core/v0"
	kind:       "StatusProbe"

	metadata: {
		apiVersion!: #NameType                          // Example: "opm.dev/statusprobes/workload@v0"
		name!:       #NameType                          // Example: "WorkloadReady"
		fqn:         #FQNType & "\(apiVersion)#\(name)" // Example: "opm.dev/statusprobes/workload@v0#WorkloadReady"

		description?: string
		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}

	// Input parameters for the probe (to be filled by the module developer)
	// Example: { resourceName: "frontend" }
	#params: {...}

	// Runtime Context (injected by the controller at runtime)
	// This defines the contract for what data is available to the logic.
	context: {
		// Map of all deployed resources (live state)
		// Key matches the resource ID in the deployment
		outputs: [string]: {...}

		// The concrete values used for the deployment
		values: {...}
	}

	// The Result Logic (Native CUE)
	// The controller unifies the live 'context' into this definition,
	// and reads the 'result' field to determine status.
	result: {
		healthy!: bool
		message?: string
		details?: [string]: bool | int | string
	}

	// Helper to expose the spec (OpenAPI compatibility)
	#spec!: (strings.ToCamel(metadata.name)): #params
})

#StatusProbeMap: [string]: #StatusProbe

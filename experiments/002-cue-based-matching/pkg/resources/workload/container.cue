package workload

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
)

/////////////////////////////////////////////////////////////////
//// Container Resource Definition
/////////////////////////////////////////////////////////////////

#ContainerResource: close(core.#Resource & {
	metadata: {
		apiVersion:  "opm.dev/resources/workload@v0"
		name:        "Container"
		description: "A container definition for workloads"
		labels: {
			// "core.opm.dev/category": "workload"
		}
	}

	// Default values for container resource
	#defaults: #ContainerDefaults

	// OpenAPIv3-compatible schema defining the structure of the container spec
	#spec: container: schemas.#ContainerSchema
})

#Container: close(core.#Component & {
	metadata: labels: {
		"core.opm.dev/workload-type"!: "stateless" | "stateful" | "daemon" | "task" | "scheduled-task"
		...
	}

	#resources: {(#ContainerResource.metadata.fqn): #ContainerResource}
})

#ContainerDefaults: close(schemas.#ContainerSchema & {
	// Image pull policy
	imagePullPolicy: schemas.#ContainerSchema.imagePullPolicy | *"IfNotPresent"
})

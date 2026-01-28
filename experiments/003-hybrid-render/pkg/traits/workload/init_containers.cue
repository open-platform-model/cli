package workload

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
	workload_resources "test.com/experiment/pkg/resources/workload@v0"
)

/////////////////////////////////////////////////////////////////
//// InitContainers Trait Definition
/////////////////////////////////////////////////////////////////

#InitContainersTrait: close(core.#Trait & {
	metadata: {
		apiVersion:  "opm.dev/traits/workload@v0"
		name:        "InitContainers"
		description: "A trait to specify init containers for a workload"
		labels: {
			"core.opm.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	// Default values for init containers trait
	#defaults: #InitContainersDefaults

	#spec: initContainers: schemas.#InitContainersSchema
})

#InitContainers: close(core.#Component & {
	#traits: {(#InitContainersTrait.metadata.fqn): #InitContainersTrait}
})

#InitContainersDefaults: schemas.#InitContainersSchema & []

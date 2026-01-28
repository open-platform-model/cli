package workload

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
	workload_resources "test.com/experiment/pkg/resources/workload@v0"
)

/////////////////////////////////////////////////////////////////
//// SidecarContainers Trait Definition
/////////////////////////////////////////////////////////////////

#SidecarContainersTrait: close(core.#Trait & {
	metadata: {
		apiVersion:  "opm.dev/traits/workload@v0"
		name:        "SidecarContainers"
		description: "A trait to specify sidecar containers for a workload"
		labels: {
			"core.opm.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	// Default values for sidecar containers trait
	#defaults: #SidecarContainersDefaults

	#spec: sidecarContainers: schemas.#SidecarContainersSchema
})

#SidecarContainers: close(core.#Component & {
	#traits: {(#SidecarContainersTrait.metadata.fqn): #SidecarContainersTrait}
})

#SidecarContainersDefaults: schemas.#SidecarContainersSchema & []

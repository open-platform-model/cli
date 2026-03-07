package workload

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
)

/////////////////////////////////////////////////////////////////
//// SidecarContainers Trait Definition
/////////////////////////////////////////////////////////////////

#SidecarContainersTrait: prim.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/traits/workload"
		version:     "v1"
		name:        "sidecar-containers"
		description: "A trait to specify sidecar containers for a workload"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	#defaults: #SidecarContainersDefaults

	spec: close({sidecarContainers: [...schemas.#ContainerSchema]})
}

#SidecarContainers: component.#Component & {
	#traits: {(#SidecarContainersTrait.metadata.fqn): #SidecarContainersTrait}
}

#SidecarContainersDefaults: schemas.#ContainerSchema & {}

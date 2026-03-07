package workload

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
)

/////////////////////////////////////////////////////////////////
//// Sizing Trait Definition
/////////////////////////////////////////////////////////////////

#SizingTrait: prim.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/traits/workload"
		version:     "v1"
		name:        "sizing"
		description: "A trait to specify vertical sizing behavior for a workload"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	#defaults: #SizingDefaults

	spec: close({sizing: schemas.#SizingSchema})
}

#Sizing: component.#Component & {
	#traits: {(#SizingTrait.metadata.fqn): #SizingTrait}
}

#SizingDefaults: schemas.#SizingSchema

package workload

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
)

/////////////////////////////////////////////////////////////////
//// Scaling Trait Definition
/////////////////////////////////////////////////////////////////

#ScalingTrait: prim.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/traits/workload"
		version:     "v1"
		name:        "scaling"
		description: "A trait to specify scaling behavior for a workload"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	#defaults: #ScalingDefaults

	spec: close({scaling: schemas.#ScalingSchema})
}

#Scaling: component.#Component & {
	#traits: {(#ScalingTrait.metadata.fqn): #ScalingTrait}
}

#ScalingDefaults: schemas.#ScalingSchema

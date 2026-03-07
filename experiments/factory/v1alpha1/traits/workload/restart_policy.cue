package workload

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
)

/////////////////////////////////////////////////////////////////
//// RestartPolicy Trait Definition
/////////////////////////////////////////////////////////////////

#RestartPolicyTrait: prim.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/traits/workload"
		version:     "v1"
		name:        "restart-policy"
		description: "A trait to specify the restart policy for a workload"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	#defaults: #RestartPolicyDefaults

	spec: close({restartPolicy: schemas.#RestartPolicySchema})
}

#RestartPolicy: component.#Component & {
	#traits: {(#RestartPolicyTrait.metadata.fqn): #RestartPolicyTrait}
}

#RestartPolicyDefaults: schemas.#RestartPolicySchema & {"Always"}

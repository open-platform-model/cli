package workload

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
)

/////////////////////////////////////////////////////////////////
//// DisruptionBudget Trait Definition
/////////////////////////////////////////////////////////////////

#DisruptionBudgetTrait: prim.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/traits/workload"
		version:     "v1"
		name:        "disruption-budget"
		description: "Availability constraints during voluntary disruptions"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	#defaults: #DisruptionBudgetDefaults

	spec: close({disruptionBudget: schemas.#DisruptionBudgetSchema})
}

#DisruptionBudget: component.#Component & {
	#traits: {(#DisruptionBudgetTrait.metadata.fqn): #DisruptionBudgetTrait}
}

#DisruptionBudgetDefaults: schemas.#DisruptionBudgetSchema & {
	maxUnavailable: 1
}

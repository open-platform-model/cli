package network

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
)

/////////////////////////////////////////////////////////////////
//// Expose Trait Definition
/////////////////////////////////////////////////////////////////

#ExposeTrait: prim.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/traits/network"
		version:     "v1"
		name:        "expose"
		description: "A trait to expose a workload via a service"
		labels: {
			"trait.opmodel.dev/category": "network"
		}
	}

	appliesTo: [workload_resources.#ContainerResource] // Full CUE reference (not FQN string)

	// Default values for expose trait
	#defaults: #ExposeDefaults

	spec: close({expose: schemas.#ExposeSchema})
}

#Expose: component.#Component & {
	#traits: {(#ExposeTrait.metadata.fqn): #ExposeTrait}
}

#ExposeDefaults: schemas.#ExposeSchema & {
	type: "ClusterIP"
}

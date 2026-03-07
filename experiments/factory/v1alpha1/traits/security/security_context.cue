package security

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
)

/////////////////////////////////////////////////////////////////
//// SecurityContext Trait Definition
/////////////////////////////////////////////////////////////////

#SecurityContextTrait: prim.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/traits/security"
		version:     "v1"
		name:        "security-context"
		description: "Container and pod-level security constraints"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	#defaults: #SecurityContextDefaults

	spec: close({securityContext: schemas.#SecurityContextSchema})
}

#SecurityContext: component.#Component & {
	#traits: {(#SecurityContextTrait.metadata.fqn): #SecurityContextTrait}
}

#SecurityContextDefaults: schemas.#SecurityContextSchema & {
	runAsNonRoot:             true
	allowPrivilegeEscalation: false
}

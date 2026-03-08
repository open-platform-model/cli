package security

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
)

/////////////////////////////////////////////////////////////////
//// Role Resource Definition
/////////////////////////////////////////////////////////////////

#RoleResource: prim.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/resources/security"
		version:     "v1"
		name:        "role"
		description: "An RBAC Role definition with rules and CUE-referenced subjects"
		labels: {
			"resource.opmodel.dev/category": "security"
		}
	}

	// Default values for Role resource
	#defaults: #RoleDefaults

	// OpenAPIv3-compatible schema defining the structure of the Role spec
	spec: close({role: schemas.#RoleSchema})
}

#Role: component.#Component & {
	#resources: {(#RoleResource.metadata.fqn): #RoleResource}
}

#RoleDefaults: schemas.#RoleSchema & {
	scope: "namespace"
}

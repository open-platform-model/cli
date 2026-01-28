package network

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
)

/////////////////////////////////////////////////////////////////
//// Expose Resource Definition
/////////////////////////////////////////////////////////////////

#ExposeResource: close(core.#Resource & {
	metadata: {
		apiVersion:  "opm.dev/resources/network@v0"
		name:        "Expose"
		description: "A resource to expose a network service"
		labels: {
			"core.opm.dev/category": "network"
		}
	}

	// Default values for expose resource
	#defaults: #ExposeDefaults

	#spec: expose: schemas.#ExposeSchema
})

#Expose: close(core.#Component & {
	#resources: {(#ExposeResource.metadata.fqn): #ExposeResource}
})

#ExposeDefaults: close(schemas.#ExposeSchema & {
	// Default service type
	type: *"ClusterIP" | "NodePort" | "LoadBalancer"
})

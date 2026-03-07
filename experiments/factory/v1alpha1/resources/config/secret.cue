package config

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
)

/////////////////////////////////////////////////////////////////
//// Secrets Resource Definition
/////////////////////////////////////////////////////////////////

#SecretsResource: prim.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/resources/config"
		version:     "v1"
		name:        "secrets"
		description: "A Secret definition for sensitive configuration"
		labels: {
			"resource.opmodel.dev/category": "config"
		}
	}

	// Default values for Secrets resource
	#defaults: #SecretsDefaults

	// OpenAPIv3-compatible schema defining the structure of the Secrets spec
	spec: close({secrets: [secretName=string]: schemas.#SecretSchema & {name: string | *secretName}})
}

#Secrets: component.#Component & {
	metadata: annotations: {
		"transformer.opmodel.dev/list-output": true
	}

	#resources: {(#SecretsResource.metadata.fqn): #SecretsResource}
}

#SecretsDefaults: {
	type: string | *"Opaque"
}

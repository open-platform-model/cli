package security

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
)

/////////////////////////////////////////////////////////////////
//// Encryption Trait Definition
/////////////////////////////////////////////////////////////////

#EncryptionConfigTrait: prim.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/traits/security"
		version:     "v1"
		name:        "encryption"
		description: "Enforces encryption requirements"
		labels: {
			"trait.opmodel.dev/category": "security"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	// Default values for encryption policy
	#defaults: #EncryptionConfigDefaults

	spec: close({encryption: schemas.#EncryptionConfigSchema})
}

#EncryptionConfig: component.#Component & {
	#traits: {(#EncryptionConfigTrait.metadata.fqn): #EncryptionConfigTrait}
}

#EncryptionConfigDefaults: schemas.#EncryptionConfigSchema & {
	atRest:    true
	inTransit: true
}

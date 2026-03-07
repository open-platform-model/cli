package extension

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
)

/////////////////////////////////////////////////////////////////
//// CRDs Resource Definition
/////////////////////////////////////////////////////////////////

#CRDsResource: prim.#Resource & {
	metadata: {
		modulePath:  "opmodel.dev/resources/extension"
		version:     "v1"
		name:        "crds"
		description: "One or more CustomResourceDefinitions to deploy to the cluster"
		labels: {
			"resource.opmodel.dev/category": "extension"
		}
	}

	// Default values for CRDs resource
	#defaults: #CRDsDefaults

	// Map of CRDs keyed by a stable identifier (typically "<plural>.<group>")
	spec: close({crds: [name=string]: schemas.#CRDSchema})
}

#CRDs: component.#Component & {
	metadata: annotations: {
		"transformer.opmodel.dev/list-output": true
	}

	#resources: {(#CRDsResource.metadata.fqn): #CRDsResource}
}

#CRDsDefaults: {
	scope: *"Namespaced" | "Cluster"
}

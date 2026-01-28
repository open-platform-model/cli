package workload

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
	workload_resources "test.com/experiment/pkg/resources/workload@v0"
)

/////////////////////////////////////////////////////////////////
//// ResourceLimit Trait Definition
/////////////////////////////////////////////////////////////////

#ResourceLimitTrait: close(core.#Trait & {
	metadata: {
		apiVersion:  "opm.dev/traits/workload@v0"
		name:        "ResourceLimit"
		description: "A trait to specify resource limits for a workload"
		labels: {
			"core.opm.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	// Default values for resource limit trait
	#defaults: #ResourceLimitDefaults

	#spec: resourceLimit: schemas.#ResourceLimitSchema
})

#ResourceLimit: close(core.#Component & {
	#traits: {(#ResourceLimitTrait.metadata.fqn): #ResourceLimitTrait}
})

#ResourceLimitDefaults: close(schemas.#ResourceLimitSchema & {
	cpu?: schemas.#ResourceLimitSchema.cpu & {
		request!: string | *"100m"
	}
	memory?: schemas.#ResourceLimitSchema.memory & {
		request!: string | *"128Mi"
	}
})

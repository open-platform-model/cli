package workload

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
)

/////////////////////////////////////////////////////////////////
//// JobConfig Trait Definition
/////////////////////////////////////////////////////////////////

#JobConfigTrait: prim.#Trait & {
	metadata: {
		modulePath:  "opmodel.dev/traits/workload"
		version:     "v1"
		name:        "job-config"
		description: "A trait to configure Job-specific settings for task workloads"
		labels: {
			"trait.opmodel.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	#defaults: #JobConfigDefaults

	spec: close({jobConfig: schemas.#JobConfigSchema})
}

#JobConfig: component.#Component & {
	#traits: {(#JobConfigTrait.metadata.fqn): #JobConfigTrait}
}

#JobConfigDefaults: schemas.#JobConfigSchema & {
	completions:             1
	parallelism:             1
	backoffLimit:            6
	activeDeadlineSeconds:   300
	ttlSecondsAfterFinished: 100
}

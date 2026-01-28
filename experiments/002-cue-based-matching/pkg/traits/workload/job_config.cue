package workload

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
	workload_resources "test.com/experiment/pkg/resources/workload@v0"
)

/////////////////////////////////////////////////////////////////
//// JobConfig Trait Definition
/////////////////////////////////////////////////////////////////

#JobConfigTrait: close(core.#Trait & {
	metadata: {
		apiVersion:  "opm.dev/traits/workload@v0"
		name:        "JobConfig"
		description: "A trait to configure Job-specific settings for task workloads"
		labels: {
			"core.opm.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	// Default values for job config trait
	#defaults: #JobConfigDefaults

	#spec: jobConfig: schemas.#JobConfigSchema
})

#JobConfig: close(core.#Component & {
	#traits: {(#JobConfigTrait.metadata.fqn): #JobConfigTrait}
})

#JobConfigDefaults: close(schemas.#JobConfigSchema & {
	completions:             1
	parallelism:             1
	backoffLimit:            6
	activeDeadlineSeconds:   300
	ttlSecondsAfterFinished: 100
})

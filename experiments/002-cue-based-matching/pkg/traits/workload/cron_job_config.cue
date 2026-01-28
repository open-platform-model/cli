package workload

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
	workload_resources "test.com/experiment/pkg/resources/workload@v0"
)

/////////////////////////////////////////////////////////////////
//// CronJobConfig Trait Definition
/////////////////////////////////////////////////////////////////

#CronJobConfigTrait: close(core.#Trait & {
	metadata: {
		apiVersion:  "opm.dev/traits/workload@v0"
		name:        "CronJobConfig"
		description: "A trait to configure CronJob-specific settings for scheduled task workloads"
		labels: {
			"core.opm.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	// Default values for cron job config trait
	#defaults: #CronJobConfigDefaults

	#spec: cronJobConfig: schemas.#CronJobConfigSchema
})

#CronJobConfig: close(core.#Component & {
	#traits: {(#CronJobConfigTrait.metadata.fqn): #CronJobConfigTrait}
})

#CronJobConfigDefaults: close(schemas.#CronJobConfigSchema & {
	concurrencyPolicy:          "Allow"
	successfulJobsHistoryLimit: 3
	failedJobsHistoryLimit:     1
})

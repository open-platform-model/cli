package workload

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
	workload_traits "opmodel.dev/traits/workload@v1"
)

/////////////////////////////////////////////////////////////////
//// ScheduledTaskWorkload Blueprint Definition
/////////////////////////////////////////////////////////////////

#ScheduledTaskWorkloadBlueprint: prim.#Blueprint & {
	metadata: {
		modulePath:  "opmodel.dev/blueprints/workload"
		version:     "v1"
		name:        "scheduled-task-workload"
		description: "A scheduled task workload that runs on a cron schedule (CronJob)"
	}

	composedResources: [
		workload_resources.#ContainerResource,
	]

	composedTraits: [
		workload_traits.#CronJobConfigTrait,
		workload_traits.#RestartPolicyTrait,
		workload_traits.#SidecarContainersTrait,
		workload_traits.#InitContainersTrait,
	]

	spec: scheduledTaskWorkload: schemas.#ScheduledTaskWorkloadSchema
}

#ScheduledTaskWorkload: component.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "scheduled-task"
	}

	#blueprints: (#ScheduledTaskWorkloadBlueprint.metadata.fqn): #ScheduledTaskWorkloadBlueprint

	workload_resources.#Container
	workload_traits.#RestartPolicy
	workload_traits.#CronJobConfig
	workload_traits.#SidecarContainers
	workload_traits.#InitContainers

	// Override spec to propagate values from scheduledTaskWorkload
	spec: {
		scheduledTaskWorkload: schemas.#ScheduledTaskWorkloadSchema
		container:             spec.scheduledTaskWorkload.container
		if spec.scheduledTaskWorkload.restartPolicy != _|_ {
			restartPolicy: spec.scheduledTaskWorkload.restartPolicy
		}
		cronJobConfig: spec.scheduledTaskWorkload.cronJobConfig
		if spec.scheduledTaskWorkload.sidecarContainers != _|_ {
			sidecarContainers: spec.scheduledTaskWorkload.sidecarContainers
		}
		if spec.scheduledTaskWorkload.initContainers != _|_ {
			initContainers: spec.scheduledTaskWorkload.initContainers
		}
	}
}

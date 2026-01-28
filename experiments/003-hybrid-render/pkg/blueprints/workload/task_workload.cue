package workload

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
	workload_resources "test.com/experiment/pkg/resources/workload@v0"
	workload_traits "test.com/experiment/pkg/traits/workload@v0"
)

/////////////////////////////////////////////////////////////////
//// TaskWorkload Blueprint Definition
/////////////////////////////////////////////////////////////////

#TaskWorkloadBlueprint: close(core.#Blueprint & {
	metadata: {
		apiVersion:  "opm.dev/blueprints/core@v0"
		name:        "TaskWorkload"
		description: "A one-time task workload that runs to completion (Job)"
		labels: {
			"core.opm.dev/category":      "workload"
			"core.opm.dev/workload-type": "task"
		}
	}

	composedResources: [
		workload_resources.#ContainerResource,
	]

	composedTraits: [
		workload_traits.#JobConfigTrait,
		workload_traits.#RestartPolicyTrait,
		workload_traits.#SidecarContainersTrait,
		workload_traits.#InitContainersTrait,
	]

	#spec: taskWorkload: schemas.#TaskWorkloadSchema
})

#TaskWorkload: close(core.#Component & {
	#blueprints: (#TaskWorkloadBlueprint.metadata.fqn): #TaskWorkloadBlueprint

	workload_resources.#Container
	workload_traits.#RestartPolicy
	workload_traits.#JobConfig
	workload_traits.#SidecarContainers
	workload_traits.#InitContainers

	spec: {
		taskWorkload: schemas.#TaskWorkloadSchema
		container:    taskWorkload.container
		if taskWorkload.restartPolicy != _|_ {
			restartPolicy: taskWorkload.restartPolicy
		}
		if taskWorkload.jobConfig != _|_ {
			jobConfig: taskWorkload.jobConfig
		}
		if taskWorkload.sidecarContainers != _|_ {
			sidecarContainers: taskWorkload.sidecarContainers
		}
		if taskWorkload.initContainers != _|_ {
			initContainers: taskWorkload.initContainers
		}
	}
})

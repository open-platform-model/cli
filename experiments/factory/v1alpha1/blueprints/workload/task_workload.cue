package workload

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
	workload_traits "opmodel.dev/traits/workload@v1"
)

/////////////////////////////////////////////////////////////////
//// TaskWorkload Blueprint Definition
/////////////////////////////////////////////////////////////////

#TaskWorkloadBlueprint: prim.#Blueprint & {
	metadata: {
		modulePath:  "opmodel.dev/blueprints/workload"
		version:     "v1"
		name:        "task-workload"
		description: "A one-time task workload that runs to completion (Job)"
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

	spec: taskWorkload: schemas.#TaskWorkloadSchema
}

#TaskWorkload: component.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "task"
	}

	#blueprints: (#TaskWorkloadBlueprint.metadata.fqn): #TaskWorkloadBlueprint

	workload_resources.#Container
	workload_traits.#RestartPolicy
	workload_traits.#JobConfig
	workload_traits.#SidecarContainers
	workload_traits.#InitContainers

	// Override spec to propagate values from taskWorkload
	spec: {
		taskWorkload: schemas.#TaskWorkloadSchema
		container:    spec.taskWorkload.container
		if spec.taskWorkload.restartPolicy != _|_ {
			restartPolicy: spec.taskWorkload.restartPolicy
		}
		if spec.taskWorkload.jobConfig != _|_ {
			jobConfig: spec.taskWorkload.jobConfig
		}
		if spec.taskWorkload.sidecarContainers != _|_ {
			sidecarContainers: spec.taskWorkload.sidecarContainers
		}
		if spec.taskWorkload.initContainers != _|_ {
			initContainers: spec.taskWorkload.initContainers
		}
	}
}

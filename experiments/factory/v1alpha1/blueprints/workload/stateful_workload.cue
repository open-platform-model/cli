package workload

import (
	prim "opmodel.dev/core/primitives@v1"
	component "opmodel.dev/core/component@v1"
	schemas "opmodel.dev/schemas@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
	workload_traits "opmodel.dev/traits/workload@v1"
	storage_resources "opmodel.dev/resources/storage@v1"
)

/////////////////////////////////////////////////////////////////
//// StatefulWorkload Blueprint Definition
/////////////////////////////////////////////////////////////////

#StatefulWorkloadBlueprint: prim.#Blueprint & {
	metadata: {
		modulePath:  "opmodel.dev/blueprints/workload"
		version:     "v1"
		name:        "stateful-workload"
		description: "A stateful workload with stable identity and persistent storage requirements"
	}

	composedResources: [
		workload_resources.#ContainerResource,
		storage_resources.#VolumeResource,
	]

	composedTraits: [
		workload_traits.#ScalingTrait,
		workload_traits.#RestartPolicyTrait,
		workload_traits.#UpdateStrategyTrait,
		workload_traits.#SidecarContainersTrait,
		workload_traits.#InitContainersTrait,
	]

	spec: statefulWorkload: schemas.#StatefulWorkloadSchema
}

#StatefulWorkload: component.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type": "stateful"
	}

	#blueprints: (#StatefulWorkloadBlueprint.metadata.fqn): #StatefulWorkloadBlueprint

	workload_resources.#Container
	workload_traits.#Scaling
	workload_traits.#RestartPolicy
	workload_traits.#UpdateStrategy
	workload_traits.#SidecarContainers
	workload_traits.#InitContainers
	storage_resources.#Volumes

	// Override spec to propagate values from statefulWorkload
	spec: {
		statefulWorkload: schemas.#StatefulWorkloadSchema
		container:        spec.statefulWorkload.container
		if spec.statefulWorkload.scaling != _|_ {
			scaling: spec.statefulWorkload.scaling
		}
		if spec.statefulWorkload.restartPolicy != _|_ {
			restartPolicy: spec.statefulWorkload.restartPolicy
		}
		if spec.statefulWorkload.updateStrategy != _|_ {
			updateStrategy: spec.statefulWorkload.updateStrategy
		}
		if spec.statefulWorkload.sidecarContainers != _|_ {
			sidecarContainers: spec.statefulWorkload.sidecarContainers
		}
		if spec.statefulWorkload.initContainers != _|_ {
			initContainers: spec.statefulWorkload.initContainers
		}
		if spec.statefulWorkload.volumes != _|_ {
			volumes: spec.statefulWorkload.volumes
		}
	}
}

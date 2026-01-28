package workload

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
	workload_resources "test.com/experiment/pkg/resources/workload@v0"
	workload_traits "test.com/experiment/pkg/traits/workload@v0"
)

/////////////////////////////////////////////////////////////////
//// DaemonWorkload Blueprint Definition
/////////////////////////////////////////////////////////////////

#DaemonWorkloadBlueprint: close(core.#Blueprint & {
	metadata: {
		apiVersion:  "opm.dev/blueprints/core@v0"
		name:        "DaemonWorkload"
		description: "A daemon workload that runs on all (or selected) nodes in a cluster"
		labels: {
			"core.opm.dev/category":      "workload"
			"core.opm.dev/workload-type": "daemon"
		}
	}

	composedResources: [
		workload_resources.#ContainerResource,
	]

	composedTraits: [
		workload_traits.#RestartPolicyTrait,
		workload_traits.#UpdateStrategyTrait,
		workload_traits.#HealthCheckTrait,
		workload_traits.#SidecarContainersTrait,
		workload_traits.#InitContainersTrait,
	]

	#spec: daemonWorkload: schemas.#DaemonWorkloadSchema
})

#DaemonWorkload: close(core.#Component & {
	#blueprints: (#DaemonWorkloadBlueprint.metadata.fqn): #DaemonWorkloadBlueprint

	workload_resources.#Container
	workload_traits.#RestartPolicy
	workload_traits.#UpdateStrategy
	workload_traits.#HealthCheck
	workload_traits.#SidecarContainers
	workload_traits.#InitContainers

	spec: {
		daemonWorkload: schemas.#DaemonWorkloadSchema
		container:      daemonWorkload.container
		if daemonWorkload.restartPolicy != _|_ {
			restartPolicy: daemonWorkload.restartPolicy
		}
		if daemonWorkload.updateStrategy != _|_ {
			updateStrategy: daemonWorkload.updateStrategy
		}
		if daemonWorkload.healthCheck != _|_ {
			healthCheck: daemonWorkload.healthCheck
		}
		if daemonWorkload.sidecarContainers != _|_ {
			sidecarContainers: daemonWorkload.sidecarContainers
		}
		if daemonWorkload.initContainers != _|_ {
			initContainers: daemonWorkload.initContainers
		}
	}
})

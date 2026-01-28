package workload

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
	workload_resources "test.com/experiment/pkg/resources/workload@v0"
)

/////////////////////////////////////////////////////////////////
//// Replicas Trait Definition
/////////////////////////////////////////////////////////////////

#ReplicasTrait: close(core.#Trait & {
	metadata: {
		apiVersion:  "opm.dev/traits/scaling@v0"
		name:        "Replicas"
		description: "A trait to specify the number of replicas for a workload"
		labels: {
			"core.opm.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource] // Full CUE reference (not FQN string)

	// Default values for replicas trait
	#defaults: #ReplicasDefaults

	#spec: replicas: schemas.#ReplicasSchema
})

#Replicas: close(core.#Component & {
	#traits: {(#ReplicasTrait.metadata.fqn): #ReplicasTrait}
})

#ReplicasDefaults: schemas.#ReplicasSchema & int | *1

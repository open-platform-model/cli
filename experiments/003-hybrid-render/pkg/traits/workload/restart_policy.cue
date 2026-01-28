package workload

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
	workload_resources "test.com/experiment/pkg/resources/workload@v0"
)

/////////////////////////////////////////////////////////////////
//// RestartPolicy Trait Definition
/////////////////////////////////////////////////////////////////

#RestartPolicyTrait: close(core.#Trait & {
	metadata: {
		apiVersion:  "opm.dev/traits/workload@v0"
		name:        "RestartPolicy"
		description: "A trait to specify the restart policy for a workload"
		labels: {
			"core.opm.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	// Default values for restart policy trait
	#defaults: #RestartPolicyDefaults

	#spec: restartPolicy: schemas.#RestartPolicySchema
})

#RestartPolicy: close(core.#Component & {
	#traits: {(#RestartPolicyTrait.metadata.fqn): #RestartPolicyTrait}
})

#RestartPolicyDefaults: schemas.#RestartPolicySchema & "Always"

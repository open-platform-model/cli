package workload

import (
	core "test.com/experiment/pkg/core@v0"
	schemas "test.com/experiment/pkg/schemas@v0"
	workload_resources "test.com/experiment/pkg/resources/workload@v0"
)

/////////////////////////////////////////////////////////////////
//// HealthCheck Trait Definition
/////////////////////////////////////////////////////////////////

#HealthCheckTrait: close(core.#Trait & {
	metadata: {
		apiVersion:  "opm.dev/traits/workload@v0"
		name:        "HealthCheck"
		description: "A trait to specify liveness and readiness probes for a workload"
		labels: {
			"core.opm.dev/category": "workload"
		}
	}

	appliesTo: [workload_resources.#ContainerResource]

	// Default values for health check trait
	#defaults: #HealthCheckDefaults

	#spec: healthCheck: schemas.#HealthCheckSchema
})

#HealthCheck: close(core.#Component & {
	#traits: {(#HealthCheckTrait.metadata.fqn): #HealthCheckTrait}
})

#HealthCheckDefaults: close(schemas.#HealthCheckSchema & {})

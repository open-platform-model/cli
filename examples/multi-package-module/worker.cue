// Worker component definition
package main

import (
	resources_workload "opmodel.dev/resources/workload@v0"
	traits_workload "opmodel.dev/traits/workload@v0"
)

// Worker component (background job processor)
#components: worker: {
	resources_workload.#Container
	traits_workload.#Scaling

	metadata: {
		name: "worker"
		labels: "core.opmodel.dev/workload-type": "stateless"
	}

	spec: {
		scaling: count: #config.worker.replicas

		container: {
			name:  "worker"
			image: #config.worker.image
			env: {
				WORKER_MODE: {
					name:  "WORKER_MODE"
					value: "background"
				}
			}
		}
	}
}

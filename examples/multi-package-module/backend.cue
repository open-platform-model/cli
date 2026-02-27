// Backend component definition
package main

import (
	resources_workload "opmodel.dev/resources/workload@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
)

// Backend component
#components: backend: {
	resources_workload.#Container
	traits_workload.#Scaling
	traits_network.#Expose

	metadata: {
		name: "backend"
		labels: "core.opmodel.dev/workload-type": "stateless"
	}

	spec: {
		scaling: count: #config.backend.replicas

		container: {
			name:  "backend"
			image: #config.backend.image
			ports: http: {
				name:       "http"
				targetPort: #config.backend.port
				protocol:   "TCP"
			}
		}

		expose: {
			ports: http: container.ports.http & {
				exposedPort: #config.backend.port
			}
			type: "ClusterIP"
		}
	}
}

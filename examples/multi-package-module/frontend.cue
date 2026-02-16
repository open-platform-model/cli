// Frontend component definition
// This file is part of the same package but in a separate file for organization.
package main

import (
	resources_workload "opmodel.dev/resources/workload@v0"
	traits_workload "opmodel.dev/traits/workload@v0"
	traits_network "opmodel.dev/traits/network@v0"
)

// Frontend component
#components: frontend: {
	resources_workload.#Container
	traits_workload.#Scaling
	traits_network.#Expose

	metadata: {
		name: "frontend"
		labels: "core.opmodel.dev/workload-type": "stateless"
	}

	spec: {
		scaling: count: #config.frontend.replicas

		container: {
			name:  "frontend"
			image: #config.frontend.image
			ports: http: {
				name:       "http"
				targetPort: 80
				protocol:   "TCP"
			}
			env: {
				BACKEND_URL: {
					name:  "BACKEND_URL"
					value: "http://backend:\(#config.backend.port)"
				}
			}
		}

		expose: {
			ports: http: container.ports.http & {
				exposedPort: #config.frontend.port
			}
			type: "ClusterIP"
		}
	}
}

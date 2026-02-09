// Components defines the workloads for this module.
// Separated from module.cue for better maintainability.
package main

import (
	resources_workload "opmodel.dev/resources/workload@v0"
	traits_workload "opmodel.dev/traits/workload@v0"
	traits_network "opmodel.dev/traits/network@v0"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {
	// Web frontend component
	web: {
		resources_workload.#Container
		traits_workload.#Replicas
		traits_network.#Expose

		metadata: labels: "core.opmodel.dev/workload-type": "stateless"

		spec: {
			container: {
				name:  "web"
				image: #config.web.image
				ports: http: {
					targetPort: 80
				}
				env: apiUrl: {
					name:  "API_URL"
					value: "http://api:\(#config.api.port)"
				}
			}
			replicas: #config.web.replicas
			expose: {
				ports: http: container.ports.http & {
					exposedPort: #config.web.port
				}
				type: "ClusterIP"
			}
		}
	}

	// API backend component
	api: {
		resources_workload.#Container
		traits_workload.#Replicas

		metadata: labels: "core.opmodel.dev/workload-type": "stateless"

		spec: {
			container: {
				name:  "api"
				image: #config.api.image
				ports: http: {
					targetPort: #config.api.port
				}
			}
			replicas: #config.api.replicas
		}
	}
}

// Components defines the workloads for this module.
// Separated from module.cue for better maintainability.
package main

import (
	resources_workload "opmodel.dev/resources/workload@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {
	// Web frontend component
	web: {
		resources_workload.#Container
		traits_workload.#Scaling
		traits_network.#Expose

		metadata: {
			name:   "web"
			labels: "core.opmodel.dev/workload-type": "stateless"
		}

		spec: {
			scaling: count: #config.web.replicas

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
		traits_workload.#Scaling

		metadata: {
			name:   "api"
			labels: "core.opmodel.dev/workload-type": "stateless"
		}

		spec: {
			scaling: count: #config.api.replicas

			container: {
				name:  "api"
				image: #config.api.image
				ports: http: {
					targetPort: #config.api.port
				}
			}
		}
	}
}

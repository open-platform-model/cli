package fixture

import (
	resources_workload "opmodel.dev/opm/v1alpha1/resources/workload@v1"
	traits_workload "opmodel.dev/opm/v1alpha1/traits/workload@v1"
	traits_network "opmodel.dev/opm/v1alpha1/traits/network@v1"
)

#components: {
	app: {
		resources_workload.#Container
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_network.#Expose

		metadata: labels: "core.opmodel.dev/workload-type": "stateless"

		spec: {
			scaling: count: #config.replicas
			restartPolicy: "Always"

			container: {
				name:  "app"
				image: #config.image
				ports: http: {
					targetPort: #config.port
					protocol:   "TCP"
				}
				if #config.resources != _|_ {
					resources: #config.resources
				}
			}

			expose: {
				type: #config.serviceType
				ports: http: {
					targetPort:  #config.port
					exposedPort: #config.port
					protocol:    "TCP"
				}
			}
		}
	}
}

// Components for the integration-test module. The api component is gated on
// #config.api.enabled so the test can verify the prune path by re-applying
// with `enabled: false`.
package itest

import (
	resources_workload "opmodel.dev/opm/v1alpha1/resources/workload@v1"
	traits_workload "opmodel.dev/opm/v1alpha1/traits/workload@v1"
	traits_network "opmodel.dev/opm/v1alpha1/traits/network@v1"
)

#components: {
	web: {
		metadata: labels: "core.opmodel.dev/workload-type": "stateless"

		resources_workload.#Container
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_workload.#UpdateStrategy
		traits_network.#Expose

		spec: {
			restartPolicy: "Always"
			updateStrategy: type: "RollingUpdate"
			container: {
				name:  "web"
				image: #config.web.image
				ports: http: targetPort: 80
			}
			scaling: count: #config.web.scaling
			expose: {
				ports: http: container.ports.http & {exposedPort: #config.web.port}
				type: "ClusterIP"
			}
		}
	}

	if #config.api.enabled {
		api: {
			metadata: labels: "core.opmodel.dev/workload-type": "stateless"

			resources_workload.#Container
			traits_workload.#Scaling
			traits_workload.#RestartPolicy
			traits_workload.#UpdateStrategy

			spec: {
				restartPolicy: "Always"
				updateStrategy: type: "RollingUpdate"
				container: {
					name:  "api"
					image: #config.api.image
					ports: http: targetPort: #config.api.port
				}
				scaling: count: #config.api.scaling
			}
		}
	}
}

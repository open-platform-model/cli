// Components for the simple module
package simple

import (
	resources_workload "opmodel.dev/resources/workload@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
)

#components: {
	web: {
		resources_workload.#Container
		traits_workload.#Scaling

		metadata: name: "web"

		spec: {
			scaling: count: #config.replicas

			container: {
				name:  "web"
				image: #config.image
			}
		}
	}
}

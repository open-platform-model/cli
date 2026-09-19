package dupidentities

import (
	bp "opmodel.dev/catalogs/opm/blueprints/v1beta1"
)

// Two distinct components declaring one object name. The deployment-transformer
// names the Deployment from the component's metadata.name, so both render
// apps/v1 Deployment <namespace>/collider and the last apply would silently
// overwrite the first.
#components: {
	first: {
		bp.#StatelessWorkload

		metadata: name: "collider"

		spec: statelessWorkload: {
			container: {
				name:  "app"
				image: #config.image
				ports: http: {name: "http", targetPort: 9898}
			}
			scaling: count: 1
			restartPolicy: "Always"
			updateStrategy: type: "RollingUpdate"
		}
	}

	second: {
		bp.#StatelessWorkload

		metadata: name: "collider"

		spec: statelessWorkload: {
			container: {
				name:  "app"
				image: #config.image
				ports: http: {name: "http", targetPort: 9898}
			}
			scaling: count: 1
			restartPolicy: "Always"
			updateStrategy: type: "RollingUpdate"
		}
	}
}

// Components for the integration-test module. The api component is gated on
// #config.api.enabled so the test can verify the prune path by re-applying
// with values_api_off.cue (web: Deployment + Service; api: Deployment only).
package module_apply_itest

import (
	bp "opmodel.dev/catalogs/opm/blueprints/workload"
	tr "opmodel.dev/catalogs/opm/traits"
)

#components: {
	web: {
		bp.#StatelessWorkload
		tr.#Expose

		metadata: {
			name: "web"
			labels: "core.opmodel.dev/workload-type": "stateless"
		}

		spec: {
			statelessWorkload: {
				container: {
					name:  "web"
					image: #config.web.image
					ports: http: {name: "http", targetPort: 80}
				}
				scaling: count: #config.web.scaling
				restartPolicy: "Always"
				updateStrategy: type: "RollingUpdate"
			}

			expose: {
				type: "ClusterIP"
				ports: http: {name: "http", targetPort: 80}
			}
		}
	}

	if #config.api.enabled {
		api: {
			bp.#StatelessWorkload

			metadata: {
				name: "api"
				labels: "core.opmodel.dev/workload-type": "stateless"
			}

			spec: {
				statelessWorkload: {
					container: {
						name:  "api"
						image: #config.api.image
						ports: http: {name: "http", targetPort: 3000}
					}
					scaling: count: #config.api.scaling
					restartPolicy: "Always"
					updateStrategy: type: "RollingUpdate"
				}
			}
		}
	}
}

// Components defines four workloads sharing one #config:
//
//   - web:    stateless frontend behind a Service, optional HTTPRoute
//   - api:    stateless backend behind a ClusterIP Service
//   - worker: stateless background loop, nothing exposed
//   - cache:  stateful store with a persistent volume and a headless Service
//
// Blueprints stamp the workload-type label that selects the workload
// transformer (Deployment for stateless, StatefulSet for stateful); traits
// attach the rest. The PodMetadata trait stamps the module version from the
// identity package onto every pod, so a running pod names the module build
// it came from.
package advanced

import (
	bp "opmodel.dev/catalogs/opm/blueprints/v1beta1"
	tr "opmodel.dev/catalogs/opm/traits/v1beta1"

	id "opmodel.dev/templates/advanced/identity"
)

// _versionLabels is shared by every component's PodMetadata attachment.
_versionLabels: "app.kubernetes.io/version": "\(id.Version)"

#components: {
	// Web frontend — stateless, exposed, optionally routed.
	web: {
		bp.#StatelessWorkload
		tr.#Expose
		tr.#PodMetadata

		// Attached only when configured: an always-attached HTTPRoute would
		// leave spec.httpRoute non-concrete and fail the render.
		if #config.httpRoute != _|_ {
			tr.#HttpRoute
		}

		metadata: name: "web"

		if #config.httpRoute != _|_ {
			spec: httpRoute: {
				hostnames: #config.httpRoute.hostnames
				rules: [{
					matches: [{
						path: {
							type:  "PathPrefix"
							value: "/"
						}
					}]
					backendPort: #config.web.port
				}]
			}
		}
		if #config.httpRoute != _|_ if #config.httpRoute.gatewayRef != _|_ {
			spec: httpRoute: gatewayRef: #config.httpRoute.gatewayRef
		}

		spec: {
			statelessWorkload: {
				container: {
					name:  "web"
					image: #config.web.image
					ports: http: {
						name:       "http"
						targetPort: #config.web.port
					}
				}
				scaling: count: #config.web.replicas
				restartPolicy: "Always"
				updateStrategy: {
					type: "RollingUpdate"
					rollingUpdate: {}
				}
			}

			expose: {
				ports: http: statelessWorkload.container.ports.http & {
					exposedPort: #config.web.port
				}
				type: "ClusterIP"
			}

			podMetadata: labels: _versionLabels
		}
	}

	// API backend — stateless, exposed inside the cluster only.
	api: {
		bp.#StatelessWorkload
		tr.#Expose
		tr.#PodMetadata

		metadata: name: "api"

		spec: {
			statelessWorkload: {
				container: {
					name:  "api"
					image: #config.api.image
					ports: http: {
						name:       "http"
						targetPort: #config.api.port
					}
					env: LOG_LEVEL: {
						name:  "LOG_LEVEL"
						value: #config.api.logLevel
					}
				}
				scaling: count: #config.api.replicas
				restartPolicy: "Always"
				updateStrategy: {
					type: "RollingUpdate"
					rollingUpdate: {}
				}
			}

			expose: {
				ports: http: statelessWorkload.container.ports.http & {
					exposedPort: #config.api.port
				}
				type: "ClusterIP"
			}

			podMetadata: labels: _versionLabels
		}
	}

	// Background worker — stateless, nothing exposed.
	worker: {
		bp.#StatelessWorkload
		tr.#PodMetadata

		metadata: name: "worker"

		spec: {
			statelessWorkload: {
				container: {
					name:  "worker"
					image: #config.worker.image
					command: ["/bin/sh", "-c"]
					args: ["while true; do echo working; sleep \(#config.worker.interval); done"]
				}
				scaling: count: 1
				restartPolicy: "Always"
			}

			podMetadata: labels: _versionLabels
		}
	}

	// Stateful cache — persistent volume, headless Service for stable per-pod
	// DNS identity.
	cache: {
		bp.#StatefulWorkload
		tr.#Expose
		tr.#PodMetadata

		metadata: name: "cache"

		spec: {
			statefulWorkload: {
				volumes: data: {
					name: "data"
					persistentClaim: {
						size:         #config.cache.storage.size
						accessMode:   "ReadWriteOnce"
						storageClass: #config.cache.storage.storageClass
					}
					readOnly: false
				}

				container: {
					name:  "cache"
					image: #config.cache.image
					ports: cache: {
						name:       "cache"
						targetPort: #config.cache.port
					}
					volumeMounts: data: statefulWorkload.volumes.data & {
						mountPath: "/data"
					}
				}
				scaling: count: 1
				restartPolicy: "Always"
			}

			expose: {
				ports: cache: statefulWorkload.container.ports.cache & {
					exposedPort: #config.cache.port
				}
				type:      "ClusterIP"
				clusterIP: "None"
			}

			podMetadata: labels: _versionLabels
		}
	}
}

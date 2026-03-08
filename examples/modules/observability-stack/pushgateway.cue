// Pushgateway component: metrics push endpoint for batch and ephemeral jobs.
// Conditional stateless Deployment that accepts pushed metrics and exposes them
// for Prometheus scraping. Useful for short-lived jobs that cannot be scraped.
package observability

import (
	resources_workload "opmodel.dev/resources/workload@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
	traits_security "opmodel.dev/traits/security@v1"
)

#components: {

	/////////////////////////////////////////////////////////////////
	//// Pushgateway - Conditional Batch Job Metrics Endpoint
	/////////////////////////////////////////////////////////////////

	if #config.pushgateway.enabled {
		pushgateway: {
			resources_workload.#Container
			traits_workload.#Scaling
			traits_workload.#RestartPolicy
			traits_network.#Expose
			traits_security.#SecurityContext

			metadata: labels: "core.opmodel.dev/workload-type": "stateless"

			spec: {
				scaling: count: 1

				restartPolicy: "Always"

				// === Main Container ===
				container: {
					name:  "pushgateway"
					image: #config.pushgateway.image

					ports: {
						http: {
							name:       "http"
							targetPort: #config.pushgateway.port
							protocol:   "TCP"
						}
					}

					args: [
						"--web.listen-address=:\(#config.pushgateway.port)",
						// Persist metrics across restarts via the lifecycle API
						"--web.enable-lifecycle",
						"--web.enable-admin-api",
					]

					if #config.pushgateway.resources != _|_ {
						resources: #config.pushgateway.resources
					}

					// === Health Checks ===
					livenessProbe: {
						httpGet: {
							path: "/-/healthy"
							port: #config.pushgateway.port
						}
						initialDelaySeconds: 10
						periodSeconds:       10
						timeoutSeconds:      5
						failureThreshold:    3
					}
					readinessProbe: {
						httpGet: {
							path: "/-/ready"
							port: #config.pushgateway.port
						}
						initialDelaySeconds: 5
						periodSeconds:       5
						timeoutSeconds:      3
						failureThreshold:    3
					}
				}

				// === Service Exposure ===
				expose: {
					ports: http: container.ports.http & {
						exposedPort: #config.pushgateway.port
					}
					type: "ClusterIP"
				}

				// === Security ===
				securityContext: {
					runAsNonRoot:             true
					runAsUser:                #config.security.runAsUser
					runAsGroup:               #config.security.runAsGroup
					readOnlyRootFilesystem:   true
					allowPrivilegeEscalation: false
					capabilities: drop: ["ALL"]
				}
			}
		}
	}
}

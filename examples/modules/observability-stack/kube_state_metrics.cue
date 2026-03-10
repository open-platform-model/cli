// Kube State Metrics component: Kubernetes object state exporter.
// Conditional stateless Deployment that queries the K8s API to generate metrics
// about the state of objects (pods, deployments, nodes, etc.).
package observability

import (
	resources_workload "opmodel.dev/resources/workload@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
	traits_security "opmodel.dev/traits/security@v1"
)

#components: {

	/////////////////////////////////////////////////////////////////
	//// Kube State Metrics - Conditional K8s Object Metrics
	/////////////////////////////////////////////////////////////////

	if #config.kubeStateMetrics.enabled {
		"kube-state-metrics": {
			resources_workload.#Container
			traits_workload.#Scaling
			traits_workload.#RestartPolicy
			traits_network.#Expose
			traits_security.#SecurityContext
			traits_security.#WorkloadIdentity

			metadata: labels: "core.opmodel.dev/workload-type": "stateless"

			spec: {
				// KSM is single-instance; sharding is handled differently
				scaling: count: 1

				restartPolicy: "Always"

				// === Main Container ===
				container: {
					name:  "kube-state-metrics"
					image: #config.kubeStateMetrics.image

					ports: {
						metrics: {
							name:       "metrics"
							targetPort: #config.kubeStateMetrics.port
							protocol:   "TCP"
						}
						telemetry: {
							name:       "telemetry"
							targetPort: #config.kubeStateMetrics.telemetryPort
							protocol:   "TCP"
						}
					}

					args: [
						"--port=\(#config.kubeStateMetrics.port)",
						"--telemetry-port=\(#config.kubeStateMetrics.telemetryPort)",
					]

					if #config.kubeStateMetrics.resources != _|_ {
						resources: #config.kubeStateMetrics.resources
					}

					// === Health Checks ===
					livenessProbe: {
						httpGet: {
							path: "/livez"
							port: #config.kubeStateMetrics.port
						}
						initialDelaySeconds: 5
						periodSeconds:       10
						timeoutSeconds:      5
						failureThreshold:    3
					}
					readinessProbe: {
						httpGet: {
							path: "/readyz"
							port: #config.kubeStateMetrics.telemetryPort
						}
						initialDelaySeconds: 5
						periodSeconds:       10
						timeoutSeconds:      5
						failureThreshold:    3
					}
				}

				// === Service Exposure ===
				expose: {
					ports: metrics: container.ports.metrics & {
						exposedPort: #config.kubeStateMetrics.port
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

				// === Workload Identity ===
				// KSM needs a ServiceAccount with cluster-wide read access
				// to query Kubernetes API for object state
				workloadIdentity: {
					name:           #config.kubeStateMetrics.serviceAccount.name
					automountToken: true
				}
			}
		}
	}
}

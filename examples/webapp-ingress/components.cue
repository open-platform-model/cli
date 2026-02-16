// Components defines the web application workload.
// Demonstrates Ingress, HPA, SecurityContext, WorkloadIdentity, and Sidecar containers.
package main

import (
	resources_workload "opmodel.dev/resources/workload@v0"
	traits_workload "opmodel.dev/traits/workload@v0"
	traits_network "opmodel.dev/traits/network@v0"
	traits_security "opmodel.dev/traits/security@v0"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	/////////////////////////////////////////////////////////////////
	//// Web - Production-Grade Stateless Application
	/////////////////////////////////////////////////////////////////

	web: {
		resources_workload.#Container
		traits_workload.#Scaling
		traits_workload.#HealthCheck
		traits_workload.#RestartPolicy
		traits_network.#Expose
		traits_network.#HttpRoute
		traits_security.#SecurityContext
		traits_security.#WorkloadIdentity
		if #config.web.sidecar.enabled {
			traits_workload.#SidecarContainers
		}

		metadata: {
			name: "web"
			labels: "core.opmodel.dev/workload-type": "stateless"
		}

		spec: {
			// Autoscaling with CPU-based metrics
			scaling: {
				count: #config.web.scaling.min // Initial replica count
				auto: {
					min: #config.web.scaling.min
					max: #config.web.scaling.max
					metrics: [{
						type: "cpu"
						target: {
							averageUtilization: #config.web.scaling.targetCPUUtilization
						}
					}]
				}
			}

			restartPolicy: "Always"

			// Health checks
			healthCheck: {
				livenessProbe: {
					httpGet: {
						path: "/healthz"
						port: #config.web.port
					}
					initialDelaySeconds: 30
					periodSeconds:       10
					timeoutSeconds:      5
					failureThreshold:    3
				}
				readinessProbe: {
					httpGet: {
						path: "/ready"
						port: #config.web.port
					}
					initialDelaySeconds: 10
					periodSeconds:       5
					timeoutSeconds:      3
					failureThreshold:    3
				}
			}

			// Container definition
			container: {
				name:  "web"
				image: #config.web.image
				ports: http: {
					name:       "http"
					targetPort: #config.web.port
					protocol:   "TCP"
				}
				env: {
					PORT: {
						name:  "PORT"
						value: "\(#config.web.port)"
					}
					LOG_LEVEL: {
						name:  "LOG_LEVEL"
						value: "info"
					}
				}
				resources: {
					requests: {
						cpu:    #config.web.resources.requests.cpu
						memory: #config.web.resources.requests.memory
					}
					limits: {
						cpu:    #config.web.resources.limits.cpu
						memory: #config.web.resources.limits.memory
					}
				}
			}

			// Service exposure
			expose: {
				ports: http: container.ports.http & {
					exposedPort: #config.web.port
				}
				type: "ClusterIP"
			}

			// Ingress routing
			httpRoute: {
				if #config.web.ingress.hostname != "" {
					hostnames: [#config.web.ingress.hostname]
				}
				rules: [{
					matches: [{
						path: {
							type:  "Prefix"
							value: #config.web.ingress.path
						}
					}]
					backendPort: #config.web.port
				}]
				if #config.web.ingress.ingressClassName != "" {
					ingressClassName: #config.web.ingress.ingressClassName
				}
				if #config.web.ingress.tls.enabled {
					tls: {
						mode: "Terminate"
						if #config.web.ingress.tls.secretName != _|_ {
							certificateRef: {
								name: #config.web.ingress.tls.secretName
							}
						}
					}
				}
			}

			// Security context
			securityContext: {
				runAsNonRoot:             true
				runAsUser:                #config.web.security.runAsUser
				runAsGroup:               #config.web.security.runAsGroup
				readOnlyRootFilesystem:   false // Set to true if app supports it
				allowPrivilegeEscalation: false
				capabilities: {
					drop: ["ALL"]
				}
			}

			// Workload identity (service account)
			workloadIdentity: {
				name:           #config.web.serviceAccount.name
				automountToken: false
			}

			// Sidecar containers (optional)
			if #config.web.sidecar.enabled {
				sidecarContainers: [{
					name:  "log-forwarder"
					image: #config.web.sidecar.image
					env: {
						SIDECAR_MODE: {
							name:  "SIDECAR_MODE"
							value: "forwarder"
						}
					}
				}]
			}
		}
	}
}

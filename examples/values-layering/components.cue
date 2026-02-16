// Components defines the web application workload.
// The component definition is environment-agnostic.
// Environment-specific behavior is driven by values.
package main

import (
	resources_workload "opmodel.dev/resources/workload@v0"
	traits_workload "opmodel.dev/traits/workload@v0"
	traits_network "opmodel.dev/traits/network@v0"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	/////////////////////////////////////////////////////////////////
	//// Web - Environment-Agnostic Stateless Application
	/////////////////////////////////////////////////////////////////

	web: {
		resources_workload.#Container
		traits_workload.#Scaling
		traits_workload.#HealthCheck
		traits_workload.#RestartPolicy
		traits_network.#Expose
		traits_network.#HttpRoute

		metadata: {
			name: "web"
			labels: {
				"core.opmodel.dev/workload-type": "stateless"
				"app.example.com/environment":    #config.environment
			}
		}

		spec: {
			scaling: count: #config.web.replicas

			restartPolicy: "Always"

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
					ENVIRONMENT: {
						name:  "ENVIRONMENT"
						value: #config.environment
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

			expose: {
				ports: http: container.ports.http & {
					exposedPort: #config.web.port
				}
				type: "ClusterIP"
			}

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
				ingressClassName: "nginx"
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
		}
	}
}

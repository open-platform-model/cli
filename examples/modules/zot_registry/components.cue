package zot_registry

import (
	"encoding/json"
	resources_workload "opmodel.dev/resources/workload@v1"
	resources_storage "opmodel.dev/resources/storage@v1"
	resources_config "opmodel.dev/resources/config@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
	traits_security "opmodel.dev/traits/security@v1"
)

#components: {
	registry: {
		resources_workload.#Container
		resources_storage.#Volumes
		resources_config.#ConfigMaps
		traits_workload.#Scaling
		traits_workload.#RestartPolicy
		traits_workload.#UpdateStrategy
		traits_workload.#GracefulShutdown
		traits_network.#Expose
		traits_security.#SecurityContext
		traits_security.#WorkloadIdentity

		// Conditional ingress
		if #config.httpRoute != _|_ {
			traits_network.#HttpRoute
		}

		metadata: {
			name: "registry"
			labels: {
				"core.opmodel.dev/workload-type": "stateful"
			}
		}

		// Build image reference based on variant
		let _repository = {
			if #config.image.variant == "full" {
				"ghcr.io/project-zot/zot"
			}
			if #config.image.variant == "minimal" {
				"ghcr.io/project-zot/zot-minimal"
			}
		}

		// Build Zot config.json from CUE
		let _zotConfig = {
			storage: {
				rootDirectory: #config.storage.rootDir

				if #config.storage.dedupe != _|_ {
					dedupe: #config.storage.dedupe
				}

				if #config.storage.gc != _|_ {
					gc:         #config.storage.gc.enabled
					gcDelay:    #config.storage.gc.delay
					gcInterval: #config.storage.gc.interval
				}
			}

			http: {
				address: #config.http.address
				port:    "\(#config.http.port)" // Zot expects string!

				if #config.auth != _|_ {
					auth: {
						htpasswd: {
							path: "/secret/htpasswd"
						}
					}

					if #config.auth.accessControl != _|_ {
						accessControl: {
							repositories: #config.auth.accessControl.repositories
							adminPolicy: {
								users: #config.auth.accessControl.adminUsers
								actions: ["read", "create", "update", "delete"]
							}
						}
					}
				}
			}

			log: {
				level: #config.log.level

				if #config.log.audit != _|_ {
					audit: #config.log.audit.enabled
				}
			}

			// Extensions (conditional)
			if #config.metrics != _|_ || #config.storage.scrub != _|_ || #config.sync != _|_ {
				extensions: {
					if #config.metrics != _|_ {
						metrics: {
							enable: #config.metrics.enabled
							prometheus: {
								path: "/metrics"
							}
						}
					}

					if #config.storage.scrub != _|_ {
						scrub: {
							enable:   #config.storage.scrub.enabled
							interval: #config.storage.scrub.interval
						}
					}

					if #config.sync != _|_ {
						sync: {
							registries: [
								for r in #config.sync.registries {
									urls:         r.urls
									onDemand:     r.onDemand
									tlsVerify:    r.tlsVerify
									pollInterval: r.pollInterval
									if r.content != _|_ {
										content: r.content
									}
								},
							]
						}
					}
				}
			}
		}

		spec: {
			// Container spec
			container: {
				name: "zot"
				image: {
					repository: _repository
					tag:        #config.image.tag
					digest:     #config.image.digest
					pullPolicy: #config.image.pullPolicy
				}

				ports: {
					api: {
						name:       "api"
						targetPort: #config.http.port
						protocol:   "TCP"
					}
				}

				env: {}

				volumeMounts: {
					data: {
						name:      "data"
						mountPath: #config.storage.rootDir
					}
					"zot-config": {
						name:      "zot-config"
						mountPath: "/etc/zot"
						readOnly:  true
					}
					if #config.auth != _|_ {
						"zot-secret": {
							name:      "zot-secret"
							mountPath: "/secret"
							readOnly:  true
						}
					}
				}

				resources: #config.resources

				// Health probes
				startupProbe: {
					httpGet: {
						path: "/startupz"
						port: #config.http.port
					}
					initialDelaySeconds: 5
					periodSeconds:       10
					failureThreshold:    3
				}

				livenessProbe: {
					httpGet: {
						path: "/livez"
						port: #config.http.port
					}
					initialDelaySeconds: 10
					periodSeconds:       10
					failureThreshold:    3
				}

				readinessProbe: {
					httpGet: {
						path: "/readyz"
						port: #config.http.port
					}
					initialDelaySeconds: 5
					periodSeconds:       5
					failureThreshold:    3
				}
			}

			// Volumes
			volumes: {
				data: {
					name: "data"
					if #config.storage.type == "pvc" {
						persistentClaim: {
							size:         #config.storage.size
							accessMode:   "ReadWriteOnce"
							storageClass: #config.storage.storageClass
						}
					}
					if #config.storage.type == "emptyDir" {
						emptyDir: {
							medium: "node"
						}
					}
				}

				"zot-config": {
					name: "zot-config"
					configMap: configMaps["zot-config"]
				}

				if #config.auth != _|_ {
					"zot-secret": {
						name: "zot-secret"
						secret: {
							from: #config.auth.htpasswd.credentials
						}
					}
				}
			}

			// ConfigMap with generated config.json
			configMaps: {
				"zot-config": {
					name: "zot-config"
					data: {
						"config.json": json.Marshal(_zotConfig)
					}
				}
			}

			// Workload traits
			scaling: {
				count: #config.replicas
			}

			restartPolicy: "Always"

			updateStrategy: {
				// Use Recreate for single-replica stateful workloads
				if #config.replicas == 1 {
					type: "Recreate"
				}
				if #config.replicas > 1 {
					type: "RollingUpdate"
					rollingUpdate: {
						maxUnavailable: 1
					}
				}
			}

			gracefulShutdown: {
				terminationGracePeriodSeconds: 30
			}

			// Network exposure
			expose: {
				type: "ClusterIP"
				ports: {
					api: container.ports.api
				}
			}

			// Optional HTTPRoute
			if #config.httpRoute != _|_ {
				httpRoute: {
					hostnames: #config.httpRoute.hostnames
					rules: [{
						matches: [{
							path: {
								type:  "Prefix"
								value: "/"
							}
						}]
						backendPort: #config.http.port
					}]
					if #config.httpRoute.tls != _|_ {
						tls: {
							mode: "Terminate"
							certificateRefs: [{
								kind: "Secret"
								name: #config.httpRoute.tls.secretName
							}]
						}
					}
					if #config.httpRoute.gatewayRef != _|_ {
						parentRefs: [{
							name:      #config.httpRoute.gatewayRef.name
							namespace: #config.httpRoute.gatewayRef.namespace
						}]
					}
				}
			}

			// Security context
			securityContext: #config.security

			// Workload identity
			workloadIdentity: {
				name:           "zot-registry"
				automountToken: false
			}
		}
	}
}

// Components defines the Jellyfin workload.
// Single stateful component with persistent config, media mounts, and health checks.
package main

import (
	resources_workload "opmodel.dev/resources/workload@v0"
	resources_storage "opmodel.dev/resources/storage@v0"
	traits_workload "opmodel.dev/traits/workload@v0"
	traits_network "opmodel.dev/traits/network@v0"
)

// #components contains component definitions.
// Components reference #config which gets resolved to concrete values at build time.
#components: {

	/////////////////////////////////////////////////////////////////
	//// Jellyfin - Stateful Media Server
	/////////////////////////////////////////////////////////////////

	jellyfin: {
		resources_workload.#Container
		resources_storage.#Volumes
		traits_workload.#Scaling
		traits_workload.#HealthCheck
		traits_workload.#RestartPolicy
		traits_network.#Expose

		metadata: name: "jellyfin"
		metadata: labels: "core.opmodel.dev/workload-type": "stateful"

		spec: {
			// Single replica - Jellyfin does not support horizontal scaling
			scaling: count: 1

			restartPolicy: "Always"

			healthCheck: {
				livenessProbe: {
					httpGet: {
						path: "/health"
						port: 8096
					}
					initialDelaySeconds: 30
					periodSeconds:       10
					timeoutSeconds:      5
					failureThreshold:    3
				}
				readinessProbe: {
					httpGet: {
						path: "/health"
						port: 8096
					}
					initialDelaySeconds: 10
					periodSeconds:       10
					timeoutSeconds:      3
					failureThreshold:    3
				}
			}

			container: {
				name:            "jellyfin"
				image:           #config.image
				imagePullPolicy: "IfNotPresent"
				ports: http: {
					name:       "http"
					targetPort: 8096
				}
				env: {
					PUID: {
						name:  "PUID"
						value: "\(#config.puid)"
					}
					PGID: {
						name:  "PGID"
						value: "\(#config.pgid)"
					}
					TZ: {
						name:  "TZ"
						value: #config.timezone
					}
					if #config.publishedServerUrl != _|_ {
						JELLYFIN_PublishedServerUrl: {
							name:  "JELLYFIN_PublishedServerUrl"
							value: #config.publishedServerUrl
						}
					}
				}
				resources: {
					requests: {
						cpu:    "500m"
						memory: "1Gi"
					}
					limits: {
						cpu:    "4000m"
						memory: "4Gi"
					}
				}
				volumeMounts: {
					config: {
						name:      "config"
						mountPath: "/config"
					}
					if #config.media != _|_ {
						for vName, lib in #config.media {
							(vName): {
								"name":    vName
								mountPath: lib.mountPath
							}
						}
					}
				}
			}

			// Expose the web UI
			expose: {
				ports: http: container.ports.http & {
					exposedPort: #config.port
				}
				type: "ClusterIP"
			}

			// Volumes: persistent config + media mounts
			volumes: {
				config: {
					name: "config"
					persistentClaim: size: #config.configStorageSize
				}
				if #config.media != _|_ {
					for name, lib in #config.media {
						(name): {
							"name": name
							if lib.type == "pvc" {
								persistentClaim: size: lib.size
							}
							if lib.type == "emptyDir" {
								emptyDir: {}
							}
						}
					}
				}
			}
		}
	}
}

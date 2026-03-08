// Alertmanager component: alert deduplication, grouping, routing, and silencing.
// Conditional StatefulSet with persistent storage for alert state and notification log.
// Configuration generated from pure CUE via encoding/yaml.
package observability

import (
	"encoding/yaml"
	resources_workload "opmodel.dev/resources/workload@v1"
	resources_storage "opmodel.dev/resources/storage@v1"
	resources_config "opmodel.dev/resources/config@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
	traits_security "opmodel.dev/traits/security@v1"
)

// _alertmanagerConfig builds the alertmanager.yml configuration as a typed CUE struct.
let _alertmanagerConfig = {
	global: {
		resolve_timeout: "5m"
	}
	route: {
		receiver:        #config.alertmanager.config.route.receiver
		group_by:        #config.alertmanager.config.route.groupBy
		group_wait:      #config.alertmanager.config.route.groupWait
		group_interval:  #config.alertmanager.config.route.groupInterval
		repeat_interval: #config.alertmanager.config.route.repeatInterval
	}
	receivers: [for r in #config.alertmanager.config.receivers {
		name: r.name
	}]
	// Inhibition rules to suppress duplicate notifications
	inhibit_rules: [{
		source_matchers: ["severity = critical"]
		target_matchers: ["severity = warning"]
		equal: ["alertname", "instance"]
	}]
}

#components: {

	/////////////////////////////////////////////////////////////////
	//// Alertmanager - Conditional Alert Routing Engine
	/////////////////////////////////////////////////////////////////

	if #config.alertmanager.enabled {
		alertmanager: {
			resources_workload.#Container
			resources_storage.#Volumes
			resources_config.#ConfigMaps
			traits_workload.#Scaling
			traits_workload.#RestartPolicy
			traits_network.#Expose
			traits_security.#SecurityContext

			metadata: labels: "core.opmodel.dev/workload-type": "stateful"

			spec: {
				scaling: count: #config.alertmanager.replicas

				restartPolicy: "Always"

				// === ConfigMaps ===
				configMaps: {
					"alertmanager-config": {
						name: "alertmanager-config"
						data: {
							"alertmanager.yml": yaml.Marshal(_alertmanagerConfig)
						}
					}
				}

				// === Main Container ===
				container: {
					name:  "alertmanager"
					image: #config.alertmanager.image

					ports: {
						http: {
							name:       "http"
							targetPort: #config.alertmanager.port
							protocol:   "TCP"
						}
					}

					args: [
						"--config.file=/etc/alertmanager/alertmanager.yml",
						"--storage.path=/data",
					]

					volumeMounts: {
						"alertmanager-data": volumes["alertmanager-data"] & {
							mountPath: "/data"
						}
						"alertmanager-config": volumes["alertmanager-config"] & {
							mountPath: "/etc/alertmanager"
							readOnly:  true
						}
					}

					if #config.alertmanager.resources != _|_ {
						resources: #config.alertmanager.resources
					}

					// === Health Checks ===
					livenessProbe: {
						httpGet: {
							path: "/-/healthy"
							port: #config.alertmanager.port
						}
						initialDelaySeconds: 10
						periodSeconds:       15
						timeoutSeconds:      5
						failureThreshold:    3
					}
					readinessProbe: {
						httpGet: {
							path: "/-/ready"
							port: #config.alertmanager.port
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
						exposedPort: #config.alertmanager.port
					}
					type: "ClusterIP"
				}

				// === Security ===
				securityContext: {
					runAsNonRoot:             true
					runAsUser:                #config.security.runAsUser
					runAsGroup:               #config.security.runAsGroup
					readOnlyRootFilesystem:   false
					allowPrivilegeEscalation: false
					capabilities: drop: ["ALL"]
				}

				// === Volumes ===
				volumes: {
					"alertmanager-data": {
						name: "alertmanager-data"

						if #config.alertmanager.storage.type == "pvc" {
							persistentClaim: {
								size: #config.alertmanager.storage.size
								if #config.alertmanager.storage.storageClass != _|_ {
									storageClass: #config.alertmanager.storage.storageClass
								}
							}
						}

						if #config.alertmanager.storage.type == "emptyDir" {
							emptyDir: {}
						}
					}

					// ConfigMap volume: mounts alertmanager.yml
					"alertmanager-config": {
						name: "alertmanager-config"
						configMap: configMaps["alertmanager-config"]
					}
				}
			}
		}
	}
}

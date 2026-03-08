// Node Exporter component: host-level metrics collection agent.
// Conditional DaemonSet that runs on every node to expose hardware and OS metrics.
// Uses container volumeMounts for host filesystem access (/proc, /sys, /).
// Note: hostPath volumes are handled by the provider/transformer at render time;
// the OPM volume schema does not model hostPath directly.
package observability

import (
	resources_workload "opmodel.dev/resources/workload@v1"
	resources_storage "opmodel.dev/resources/storage@v1"
	traits_workload "opmodel.dev/traits/workload@v1"
	traits_network "opmodel.dev/traits/network@v1"
	traits_security "opmodel.dev/traits/security@v1"
)

#components: {

	/////////////////////////////////////////////////////////////////
	//// Node Exporter - Conditional Host Metrics DaemonSet
	/////////////////////////////////////////////////////////////////

	if #config.nodeExporter.enabled {
		"node-exporter": {
			resources_workload.#Container
			resources_storage.#Volumes
			traits_workload.#RestartPolicy
			traits_workload.#UpdateStrategy
			traits_network.#Expose
			traits_security.#SecurityContext

			metadata: labels: "core.opmodel.dev/workload-type": "daemon"

			spec: {
				restartPolicy: "Always"

				updateStrategy: {
					type: "RollingUpdate"
					rollingUpdate: maxUnavailable: 1
				}

				// === Volumes ===
				// hostPath volumes for reading host-level metrics
				volumes: {
					"host-proc": {
						name: "host-proc"
						hostPath: {
							path: #config.nodeExporter.hostPaths.proc
							type: "Directory"
						}
					}
					"host-sys": {
						name: "host-sys"
						hostPath: {
							path: #config.nodeExporter.hostPaths.sys
							type: "Directory"
						}
					}
					"host-root": {
						name: "host-root"
						hostPath: {
							path: "/"
							type: "Directory"
						}
					}
				}

				// === Main Container ===
				container: {
					name:  "node-exporter"
					image: #config.nodeExporter.image

					ports: {
						metrics: {
							name:       "metrics"
							targetPort: #config.nodeExporter.port
							protocol:   "TCP"
						}
					}

					// Tell node-exporter where host filesystems are mounted
					args: [
						"--path.procfs=/host/proc",
						"--path.sysfs=/host/sys",
						"--path.rootfs=/host/root",
						"--web.listen-address=:\(#config.nodeExporter.port)",
						// Disable metrics that require privileged access or are noisy
						"--no-collector.wifi",
						"--no-collector.hwmon",
						"--collector.filesystem.mount-points-exclude=^/(dev|proc|sys|var/lib/docker/.+|var/lib/kubelet/.+)($|/)",
					]

					// Mount host filesystems as read-only for metrics collection
					volumeMounts: {
						"host-proc": volumes["host-proc"] & {
							mountPath: "/host/proc"
							readOnly:  true
						}
						"host-sys": volumes["host-sys"] & {
							mountPath: "/host/sys"
							readOnly:  true
						}
						"host-root": volumes["host-root"] & {
							mountPath: "/host/root"
							readOnly:  true
						}
					}

					if #config.nodeExporter.resources != _|_ {
						resources: #config.nodeExporter.resources
					}

					// === Health Checks ===
					livenessProbe: {
						httpGet: {
							path: "/metrics"
							port: #config.nodeExporter.port
						}
						initialDelaySeconds: 15
						periodSeconds:       20
						timeoutSeconds:      5
						failureThreshold:    3
					}
					readinessProbe: {
						httpGet: {
							path: "/metrics"
							port: #config.nodeExporter.port
						}
						initialDelaySeconds: 5
						periodSeconds:       10
						timeoutSeconds:      3
						failureThreshold:    3
					}
				}

				// === Service Exposure ===
				// ClusterIP service so Prometheus can discover and scrape all instances
				expose: {
					ports: metrics: container.ports.metrics & {
						exposedPort: #config.nodeExporter.port
					}
					type: "ClusterIP"
				}

				// === Security ===
				// Node exporter needs to read host filesystems but should not escalate
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

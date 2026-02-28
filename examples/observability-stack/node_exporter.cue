// Node Exporter component: host-level metrics collection agent.
// Conditional DaemonSet that runs on every node to expose hardware and OS metrics.
// Uses container volumeMounts for host filesystem access (/proc, /sys, /).
// Note: hostPath volumes are handled by the provider/transformer at render time;
// the OPM volume schema does not model hostPath directly.
package main

import (
	resources_workload "opmodel.dev/resources/workload@v1"
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

					// NOTE: Node exporter typically needs hostPath volume mounts for
					// /proc, /sys, and / to read host-level metrics. However, OPM's
					// #VolumeSchema does not currently support hostPath volumes.
					// In production, these would be added via provider-specific config
					// or a custom transformer extension.

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

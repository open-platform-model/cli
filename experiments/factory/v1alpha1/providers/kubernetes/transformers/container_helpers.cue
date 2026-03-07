package transformers

import (
	k8scorev1 "opmodel.dev/schemas/kubernetes/core/v1@v1"
	schemas "opmodel.dev/schemas@v1"
)

// #ToK8sContainer converts an OPM #ContainerSchema to a Kubernetes #Container.
// OPM uses struct-keyed env/ports/volumeMounts; Kubernetes expects lists.
//
// Env var dispatch:
//   value?            -> { name, value }
//   from?             -> { name, valueFrom: { secretKeyRef: ... } }
//   fieldRef?         -> { name, valueFrom: { fieldRef: ... } }
//   resourceFieldRef? -> { name, valueFrom: { resourceFieldRef: ... } }
//
// Usage:
//   (#ToK8sContainer & {"in": _container}).out
#ToK8sContainer: {
	X="in": schemas.#ContainerSchema

	out: k8scorev1.#Container & {
		name:            X.name
		image:           X.image.reference
		imagePullPolicy: X.image.pullPolicy
		if X.command != _|_ {
			command: X.command
		}
		if X.args != _|_ {
			args: X.args
		}
		if X.ports != _|_ {
			ports: [for _, p in X.ports {
				name:          p.name
				containerPort: p.targetPort
				protocol:      p.protocol
				if p.hostIP != _|_ {hostIP: p.hostIP}
				if p.hostPort != _|_ {hostPort: p.hostPort}
			}]
		}

		// Env var dispatch: map OPM source types to K8s env entries
		if X.env != _|_ {
			env: [for _, e in X.env {
				// Literal value — inline string
				if e.value != _|_ {
					name:  e.name
					value: e.value
				}

				// Secret reference — dispatch by variant
				if e.from != _|_ {
					name: e.name
					// #SecretK8sRef: use the pre-existing K8s Secret name + key
					if e.from.secretName != _|_ {
						valueFrom: secretKeyRef: {
							name: e.from.secretName
							key:  e.from.remoteKey
						}
					}

					// #SecretLiteral / #SecretEsoRef: use $secretName / $dataKey
					if e.from.secretName == _|_ {
						valueFrom: secretKeyRef: {
							name: e.from.$secretName
							key:  e.from.$dataKey
						}
					}
				}

				// Downward API — pod/container metadata
				if e.fieldRef != _|_ {
					name: e.name
					valueFrom: fieldRef: {
						fieldPath: e.fieldRef.fieldPath
						if e.fieldRef.apiVersion != _|_ {
							apiVersion: e.fieldRef.apiVersion
						}
					}
				}

				// Container resource limits/requests
				if e.resourceFieldRef != _|_ {
					name: e.name
					valueFrom: resourceFieldRef: {
						resource: e.resourceFieldRef.resource
						if e.resourceFieldRef.containerName != _|_ {
							containerName: e.resourceFieldRef.containerName
						}
						if e.resourceFieldRef.divisor != _|_ {
							divisor: e.resourceFieldRef.divisor
						}
					}
				}
			}]
		}

		// Bulk injection from ConfigMaps/Secrets
		if X.envFrom != _|_ {
			envFrom: X.envFrom
		}

		if X.resources != _|_ {
			resources: {
				if X.resources.requests != _|_ {
					requests: {
						if X.resources.requests.cpu != _|_ {
							cpu: (schemas.#NormalizeCPU & {in: X.resources.requests.cpu}).out
						}
						if X.resources.requests.memory != _|_ {
							memory: (schemas.#NormalizeMemory & {in: X.resources.requests.memory}).out
						}
					}
				}

				if X.resources.limits != _|_ {
					limits: {
						if X.resources.limits.cpu != _|_ {
							cpu: (schemas.#NormalizeCPU & {in: X.resources.limits.cpu}).out
						}
						if X.resources.limits.memory != _|_ {
							memory: (schemas.#NormalizeMemory & {in: X.resources.limits.memory}).out
						}
					}
				}
			}
		}

		// Volume mounts: extract only K8s-valid fields (strip embedded volume source data)
		if X.volumeMounts != _|_ {
			volumeMounts: [for _, vm in X.volumeMounts {
				name:      vm.name
				mountPath: vm.mountPath
				if vm.subPath != _|_ {subPath: vm.subPath}
				if vm.readOnly == true {readOnly: vm.readOnly}
			}]
		}

		if X.startupProbe != _|_ {
			startupProbe: X.startupProbe
		}
		if X.livenessProbe != _|_ {
			livenessProbe: X.livenessProbe
		}
		if X.readinessProbe != _|_ {
			readinessProbe: X.readinessProbe
		}
	}
}

// #ToK8sContainers converts a list of OPM containers to Kubernetes containers.
//
// Usage:
//   (#ToK8sContainers & {"in": _initContainers}).out
#ToK8sContainers: {
	X="in": [...schemas.#ContainerSchema]

	out: [for c in X {
		(#ToK8sContainer & {"in": c}).out
	}]
}

// #ToK8sVolumes converts OPM volumes map to Kubernetes volumes list.
// Handles all volume source types: emptyDir, persistentClaim, configMap, secret.
// For configMap and secret sources, computes the deterministic K8s resource name
// using the immutable name helpers (same helpers used by ConfigMap/Secret transformers).
//
// Usage:
//   (#ToK8sVolumes & {"in": _component.spec.volumes}).out
#ToK8sVolumes: {
	X="in": [string]: schemas.#VolumeSchema

	out: [for vName, vol in X {
		name: vol.name | *vName
		if vol.emptyDir != _|_ {
			emptyDir: {
				if vol.emptyDir.medium != _|_ if vol.emptyDir.medium == "memory" {
					medium: "Memory"
				}
				if vol.emptyDir.sizeLimit != _|_ {
					sizeLimit: vol.emptyDir.sizeLimit
				}
			}
		}
		if vol.persistentClaim != _|_ {
			persistentVolumeClaim: claimName: vol.name | *vName
		}
		if vol.configMap != _|_ {
			configMap: name: (schemas.#ImmutableName & {
				baseName:  vol.configMap.name
				data:      vol.configMap.data
				immutable: vol.configMap.immutable
			}).out
		}
		if vol.secret != _|_ {
			secret: secretName: (schemas.#SecretImmutableName & {
				baseName:  vol.secret.name
				data:      vol.secret.data
				immutable: vol.secret.immutable
			}).out
		}
		if vol.hostPath != _|_ {
			hostPath: {
				path: vol.hostPath.path
				if vol.hostPath.type != _|_ {
					type: vol.hostPath.type
				}
			}
		}
	}]
}

_testToK8sContainer: {
	// Example input container
	in: {
		name: "example-container"
		image: {
			repository: "example-image"
			tag:        "latest"
			digest:     ""
		}
		command: ["/bin/example"]
		args: ["--example-arg"]
		ports: {
			http: {
				name:       "http"
				targetPort: 8080
				protocol:   "TCP"
			}
		}
		env: {
			EXAMPLE_ENV_VAR: {
				name:  "EXAMPLE_ENV_VAR"
				value: "example-value"
			}
		}
		resources: {
			requests: {
				cpu:    "100m"
				memory: "128Mi"
			}
			limits: {
				cpu:    "200m"
				memory: "256Mi"
			}
		}
		volumeMounts: {
			exampleVolumeMount: {
				name:      "example-volume"
				mountPath: "/data/example"
			}
		}
	}

	out: (#ToK8sContainer & {"in": in}).out
}

_testToK8sContainers: {
	// Example list of input containers
	in: [
		{
			name: "example-container-1"
			image: {
				repository: "example-image-1"
				tag:        "latest"
				digest:     ""
			}
		},
		{
			name: "example-container-2"
			image: {
				repository: "example-image-2"
				tag:        "latest"
				digest:     ""
			}
		},
	]

	out: (#ToK8sContainers & {"in": in}).out
}

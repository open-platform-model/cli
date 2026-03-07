package transformers

import (
	"list"
	k8sappsv1 "opmodel.dev/schemas/kubernetes/apps/v1@v1"
	transformer "opmodel.dev/core/transformer@v1"
	workload_resources "opmodel.dev/resources/workload@v1"
	workload_traits "opmodel.dev/traits/workload@v1"
	security_traits "opmodel.dev/traits/security@v1"
	storage_resources "opmodel.dev/resources/storage@v1"
)

// StatefulsetTransformer converts stateful workload components to Kubernetes StatefulSets
#StatefulsetTransformer: transformer.#Transformer & {
	metadata: {
		modulePath:  "opmodel.dev/providers/kubernetes/transformers"
		version:     "v1"
		name:        "statefulset-transformer"
		description: "Converts stateful workload components to Kubernetes StatefulSets"

		labels: {
			"core.opmodel.dev/workload-type": "stateful"
			"core.opmodel.dev/resource-type": "statefulset"
		}
	}

	// Required label to match stateful workloads
	requiredLabels: {
		"core.opmodel.dev/workload-type": "stateful"
	}

	// Required resources - Container MUST be present
	requiredResources: {
		"opmodel.dev/resources/workload/container@v1": workload_resources.#ContainerResource
	}

	// Optional resources
	optionalResources: {
		"opmodel.dev/resources/storage/volumes@v1": storage_resources.#VolumesResource
	}

	// No required traits
	requiredTraits: {}

	// Optional traits that enhance statefulset behavior
	optionalTraits: {
		"opmodel.dev/traits/workload/scaling@v1":            workload_traits.#ScalingTrait
		"opmodel.dev/traits/workload/restart-policy@v1":     workload_traits.#RestartPolicyTrait
		"opmodel.dev/traits/workload/update-strategy@v1":    workload_traits.#UpdateStrategyTrait
		"opmodel.dev/traits/workload/health-check@v1":       workload_traits.#HealthCheckTrait
		"opmodel.dev/traits/workload/sidecar-containers@v1": workload_traits.#SidecarContainersTrait
		"opmodel.dev/traits/workload/init-containers@v1":    workload_traits.#InitContainersTrait
		"opmodel.dev/traits/security/security-context@v1":   security_traits.#SecurityContextTrait
		"opmodel.dev/traits/security/workload-identity@v1":  security_traits.#WorkloadIdentityTrait
	}

	#transform: {
		#component: _ // Unconstrained; validated by matching, not by transform signature
		#context:   transformer.#TransformerContext

		// Extract required Container resource
		_container: #component.spec.container

		// Apply defaults for optional traits
		_scalingCount: *optionalTraits["opmodel.dev/traits/workload/scaling@v1"].#defaults.count | int
		if #component.spec.scaling != _|_ if #component.spec.scaling.auto != _|_ {
			_scalingCount: #component.spec.scaling.auto.min
		}
		if #component.spec.scaling != _|_ if #component.spec.scaling.auto == _|_ {
			_scalingCount: #component.spec.scaling.count
		}

		_restartPolicy: *optionalTraits["opmodel.dev/traits/workload/restart-policy@v1"].#defaults | string
		if #component.spec.restartPolicy != _|_ {
			_restartPolicy: #component.spec.restartPolicy
		}

		// Extract update strategy with defaults
		_updateStrategy: *null | {
			if #component.spec.updateStrategy != _|_ {
				type: #component.spec.updateStrategy.type
				if #component.spec.updateStrategy.type == "RollingUpdate" {
					rollingUpdate: #component.spec.updateStrategy.rollingUpdate
				}
			}
		}

		// Build main container: base conversion via helper, unified with trait fields
		_mainContainer: (#ToK8sContainer & {"in": _container}).out

		// Build container list (main container + optional sidecars)
		_sidecarContainers: *optionalTraits["opmodel.dev/traits/workload/sidecar-containers@v1"].#defaults | [...]
		if #component.spec.sidecarContainers != _|_ {
			_sidecarContainers: #component.spec.sidecarContainers
		}

		// Extract init containers with defaults
		_initContainers: *optionalTraits["opmodel.dev/traits/workload/init-containers@v1"].#defaults | [...]
		if #component.spec.initContainers != _|_ {
			_initContainers: #component.spec.initContainers
		}

		// Build StatefulSet resource
		output: k8sappsv1.#StatefulSet & {
			apiVersion: "apps/v1"
			kind:       "StatefulSet"
			metadata: {
				name:      #component.metadata.name
				namespace: #context.#moduleReleaseMetadata.namespace
				labels:    #context.labels
				// Include component annotations if present
				if len(#context.componentAnnotations) > 0 {
					annotations: #context.componentAnnotations
				}
			}
			spec: {
				serviceName: #component.metadata.name
				replicas:    _scalingCount
				selector: matchLabels: #context.componentLabels
				template: {
					metadata: labels: #context.componentLabels
					spec: {
						_convertedSidecars: (#ToK8sContainers & {"in": _sidecarContainers}).out
						containers: list.Concat([[_mainContainer], _convertedSidecars])

						if len(_initContainers) > 0 {
							initContainers: (#ToK8sContainers & {"in": _initContainers}).out
						}

						restartPolicy: _restartPolicy

						if #component.spec.securityContext != _|_ {
							let _sc = #component.spec.securityContext
							if _sc.runAsNonRoot != _|_ || _sc.runAsUser != _|_ || _sc.runAsGroup != _|_ {
								securityContext: {
									if _sc.runAsNonRoot != _|_ {
										runAsNonRoot: _sc.runAsNonRoot
									}
									if _sc.runAsUser != _|_ {
										runAsUser: _sc.runAsUser
									}
									if _sc.runAsGroup != _|_ {
										runAsGroup: _sc.runAsGroup
									}
								}
							}
						}

						if #component.spec.workloadIdentity != _|_ {
							serviceAccountName: #component.spec.workloadIdentity.name
						}

						// Volumes: convert OPM volume specs to Kubernetes volume specs
						if #component.spec.volumes != _|_ {
							volumes: (#ToK8sVolumes & {"in": #component.spec.volumes}).out
						}
					}
				}

				if _updateStrategy != null {
					updateStrategy: _updateStrategy
				}
			}
		}
	}
}

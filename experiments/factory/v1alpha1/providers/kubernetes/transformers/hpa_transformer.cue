package transformers

import (
	transformer "opmodel.dev/core/transformer@v1"
	workload_traits "opmodel.dev/traits/workload@v1"
	k8sasv2 "opmodel.dev/schemas/kubernetes/autoscaling/v2@v1"
)

// HPATransformer converts Scaling auto config to Kubernetes HorizontalPodAutoscalers
#HPATransformer: transformer.#Transformer & {
	metadata: {
		modulePath:  "opmodel.dev/providers/kubernetes/transformers"
		version:     "v1"
		name:        "hpa-transformer"
		description: "Converts Scaling auto config to Kubernetes HorizontalPodAutoscalers"

		labels: {
			"core.opmodel.dev/trait-type":    "workload"
			"core.opmodel.dev/resource-type": "hpa"
		}
	}

	requiredLabels: {}

	requiredResources: {}
	optionalResources: {}

	// Required traits - Scaling MUST be present
	requiredTraits: {
		"opmodel.dev/traits/workload/scaling@v1": workload_traits.#ScalingTrait
	}

	optionalTraits: {}

	#transform: {
		#component: _
		#context:   transformer.#TransformerContext

		// Map workload-type label to K8s kind
		_workloadType: #component.metadata.labels["core.opmodel.dev/workload-type"]
		_kindMap: {
			stateless: "Deployment"
			stateful:  "StatefulSet"
		}
		_targetKind: _kindMap[_workloadType]

		// Only produce output when auto scaling is configured
		output: {
			if #component.spec.scaling.auto != _|_ {
				k8sasv2.#HorizontalPodAutoscaler & {
					let _auto = #component.spec.scaling.auto

					apiVersion: "autoscaling/v2"
					kind:       "HorizontalPodAutoscaler"
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
						scaleTargetRef: {
							apiVersion: "apps/v1"
							kind:       _targetKind
							name:       #component.metadata.name
						}
						minReplicas: _auto.min
						maxReplicas: _auto.max

						metrics: [
							for m in _auto.metrics {
								if m.type == "cpu" {
									type: "Resource"
									resource: {
										name:   "cpu"
										target: m.target
									}
								}
								if m.type == "memory" {
									type: "Resource"
									resource: {
										name:   "memory"
										target: m.target
									}
								}
								if m.type == "custom" {
									type: "Pods"
									pods: {
										metric: name: m.metricName
										target: m.target
									}
								}
							},
						]

						if _auto.behavior != _|_ {
							behavior: _auto.behavior
						}
					}
				}
			}
		}
	}
}

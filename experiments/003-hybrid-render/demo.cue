package hybrid

import (
	core "test.com/experiment/pkg/core"
)

// 1. Define Provider with 6 Transformers
provider: core.#Provider & {
	metadata: {
		name:       "kubernetes-test-provider"
		version:    "0.1.0"
		minVersion: "0.1.0"
		description: "Test provider with all workload transformers"
	}

	transformers: {
		"deployment": core.#Transformer & {
			metadata: {
				name:        "DeploymentTransformer"
				apiVersion:  "transformer.opm.dev/workload@v0"
				description: "Transforms stateless workloads into Kubernetes Deployments"
			}
			requiredLabels: {
				"core.opm.dev/workload-type": "stateless"
			}
			requiredResources: {
				"opm.dev/resources/workload@v0#Container": _
			}
			optionalResources: {}
			requiredTraits:    {}
			optionalTraits:    {}

		#transform: {
			#component: _
			context:    core.#TransformerContext

			let _ports = (#K8sPorts & {_component: #component}).out
			let _env = (#K8sEnv & {_component: #component}).out

			output: {
					apiVersion: "apps/v1"
					kind:       "Deployment"
					metadata: {
						name:      #component.metadata.name
						namespace: context.namespace
						labels:    context.labels
					}
					spec: {
						replicas: #component.spec.replicas
						selector: matchLabels: "app.kubernetes.io/name": #component.metadata.name
						template: {
							metadata: labels: "app.kubernetes.io/name": #component.metadata.name
							spec: {
								containers: [{
									name:            #component.spec.container.name
									image:           #component.spec.container.image
									imagePullPolicy: #component.spec.container.imagePullPolicy
									if len(_ports) > 0 {
										ports: _ports
									}
									if len(_env) > 0 {
										env: _env
									}
									if #component.spec.container.resources != _|_ {
										resources: #component.spec.container.resources
									}
								}]
								if #component.spec.restartPolicy != _|_ {
									restartPolicy: #component.spec.restartPolicy
								}
							}
						}
					}
				}
			}
		}

		"statefulset": core.#Transformer & {
			metadata: {
				name:        "StatefulSetTransformer"
				apiVersion:  "transformer.opm.dev/workload@v0"
				description: "Transforms stateful workloads into Kubernetes StatefulSets"
			}
			requiredLabels: {
				"core.opm.dev/workload-type": "stateful"
			}
			requiredResources: {
				"opm.dev/resources/workload@v0#Container": _
			}
			optionalResources: {}
			requiredTraits:    {}
			optionalTraits:    {}

		#transform: {
			#component: _
			context:    core.#TransformerContext

			let _ports = (#K8sPorts & {_component: #component}).out
			let _env = (#K8sEnv & {_component: #component}).out

			let _volumeMounts = [
					if #component.spec.container.volumeMounts != _|_
					for _, vm in #component.spec.container.volumeMounts {
						name:      vm.name
						mountPath: vm.mountPath
					},
				]

				let _volumeClaimTemplates = [
					if #component.spec.volumes != _|_
					for _, vol in #component.spec.volumes if vol.persistentClaim != _|_ {
						metadata: name: vol.name
						spec: {
							accessModes: [*vol.persistentClaim.accessMode | "ReadWriteOnce"]
							resources: requests: storage: vol.persistentClaim.size
							if vol.persistentClaim.storageClass != _|_ {
								storageClassName: vol.persistentClaim.storageClass
							}
						}
					},
				]

				output: {
					apiVersion: "apps/v1"
					kind:       "StatefulSet"
					metadata: {
						name:      #component.metadata.name
						namespace: context.namespace
						labels:    context.labels
					}
					spec: {
						serviceName: #component.metadata.name
						replicas:    #component.spec.replicas
						selector: matchLabels: "app.kubernetes.io/name": #component.metadata.name
						template: {
							metadata: labels: "app.kubernetes.io/name": #component.metadata.name
							spec: {
								containers: [{
									name:            #component.spec.container.name
									image:           #component.spec.container.image
									imagePullPolicy: #component.spec.container.imagePullPolicy
									if len(_ports) > 0 {
										ports: _ports
									}
									if len(_env) > 0 {
										env: _env
									}
									if #component.spec.container.resources != _|_ {
										resources: #component.spec.container.resources
									}
									if len(_volumeMounts) > 0 {
										volumeMounts: _volumeMounts
									}
								}]
								if #component.spec.restartPolicy != _|_ {
									restartPolicy: #component.spec.restartPolicy
								}
							}
						}
						if len(_volumeClaimTemplates) > 0 {
							volumeClaimTemplates: _volumeClaimTemplates
						}
					}
				}
			}
		}

		"daemonset": core.#Transformer & {
			metadata: {
				name:        "DaemonSetTransformer"
				apiVersion:  "transformer.opm.dev/workload@v0"
				description: "Transforms daemon workloads into Kubernetes DaemonSets"
			}
			requiredLabels: {
				"core.opm.dev/workload-type": "daemon"
			}
			requiredResources: {
				"opm.dev/resources/workload@v0#Container": _
			}
			optionalResources: {}
			requiredTraits:    {}
			optionalTraits:    {}

		#transform: {
			#component: _
			context:    core.#TransformerContext

			let _ports = (#K8sPorts & {_component: #component}).out

			output: {
				apiVersion: "apps/v1"
				kind:       "DaemonSet"
					metadata: {
						name:      #component.metadata.name
						namespace: context.namespace
						labels:    context.labels
					}
					spec: {
						selector: matchLabels: "app.kubernetes.io/name": #component.metadata.name
						template: {
							metadata: labels: "app.kubernetes.io/name": #component.metadata.name
							spec: {
								containers: [{
									name:            #component.spec.container.name
									image:           #component.spec.container.image
									imagePullPolicy: #component.spec.container.imagePullPolicy
									if len(_ports) > 0 {
										ports: _ports
									}
								}]
								if #component.spec.restartPolicy != _|_ {
									restartPolicy: #component.spec.restartPolicy
								}
							}
						}
						if #component.spec.updateStrategy != _|_ {
							updateStrategy: #component.spec.updateStrategy
						}
					}
				}
			}
		}

		"job": core.#Transformer & {
			metadata: {
				name:        "JobTransformer"
				apiVersion:  "transformer.opm.dev/workload@v0"
				description: "Transforms task workloads into Kubernetes Jobs"
			}
			requiredLabels: {
				"core.opm.dev/workload-type": "task"
			}
			requiredResources: {
				"opm.dev/resources/workload@v0#Container": _
			}
			optionalResources: {}
		requiredTraits:    {}
		optionalTraits:    {}

		#transform: {
			#component: _
			context:    core.#TransformerContext

			let _env = (#K8sEnv & {_component: #component}).out

			output: {
				apiVersion: "batch/v1"
				kind:       "Job"
					metadata: {
						name:      #component.metadata.name
						namespace: context.namespace
						labels:    context.labels
					}
					spec: {
						if #component.spec.jobConfig != _|_ {
							if #component.spec.jobConfig.completions != _|_ {
								completions: #component.spec.jobConfig.completions
							}
							if #component.spec.jobConfig.parallelism != _|_ {
								parallelism: #component.spec.jobConfig.parallelism
							}
							if #component.spec.jobConfig.backoffLimit != _|_ {
								backoffLimit: #component.spec.jobConfig.backoffLimit
							}
							if #component.spec.jobConfig.activeDeadlineSeconds != _|_ {
								activeDeadlineSeconds: #component.spec.jobConfig.activeDeadlineSeconds
							}
							if #component.spec.jobConfig.ttlSecondsAfterFinished != _|_ {
								ttlSecondsAfterFinished: #component.spec.jobConfig.ttlSecondsAfterFinished
							}
						}
						template: {
							metadata: labels: "app.kubernetes.io/name": #component.metadata.name
							spec: {
								containers: [{
									name:            #component.spec.container.name
									image:           #component.spec.container.image
									imagePullPolicy: #component.spec.container.imagePullPolicy
									if len(_env) > 0 {
										env: _env
									}
								}]
								if #component.spec.restartPolicy != _|_ {
									restartPolicy: #component.spec.restartPolicy
								}
							}
						}
					}
				}
			}
		}

		"cronjob": core.#Transformer & {
			metadata: {
				name:        "CronJobTransformer"
				apiVersion:  "transformer.opm.dev/workload@v0"
				description: "Transforms scheduled task workloads into Kubernetes CronJobs"
			}
			requiredLabels: {
				"core.opm.dev/workload-type": "scheduled-task"
			}
			requiredResources: {
				"opm.dev/resources/workload@v0#Container": _
			}
			optionalResources: {}
		requiredTraits:    {}
		optionalTraits:    {}

		#transform: {
			#component: _
			context:    core.#TransformerContext

			let _env = (#K8sEnv & {_component: #component}).out

			output: {
				apiVersion: "batch/v1"
				kind:       "CronJob"
					metadata: {
						name:      #component.metadata.name
						namespace: context.namespace
						labels:    context.labels
					}
					spec: {
						schedule: #component.spec.cronJobConfig.scheduleCron
						if #component.spec.cronJobConfig.concurrencyPolicy != _|_ {
							concurrencyPolicy: #component.spec.cronJobConfig.concurrencyPolicy
						}
						if #component.spec.cronJobConfig.startingDeadlineSeconds != _|_ {
							startingDeadlineSeconds: #component.spec.cronJobConfig.startingDeadlineSeconds
						}
						if #component.spec.cronJobConfig.successfulJobsHistoryLimit != _|_ {
							successfulJobsHistoryLimit: #component.spec.cronJobConfig.successfulJobsHistoryLimit
						}
						if #component.spec.cronJobConfig.failedJobsHistoryLimit != _|_ {
							failedJobsHistoryLimit: #component.spec.cronJobConfig.failedJobsHistoryLimit
						}
						jobTemplate: {
							metadata: labels: "app.kubernetes.io/name": #component.metadata.name
							spec: template: {
								metadata: labels: "app.kubernetes.io/name": #component.metadata.name
								spec: {
									containers: [{
										name:            #component.spec.container.name
										image:           #component.spec.container.image
										imagePullPolicy: #component.spec.container.imagePullPolicy
										if len(_env) > 0 {
											env: _env
										}
									}]
									if #component.spec.restartPolicy != _|_ {
										restartPolicy: #component.spec.restartPolicy
									}
								}
							}
						}
					}
				}
			}
		}

		"service": core.#Transformer & {
			metadata: {
				name:        "ServiceTransformer"
				apiVersion:  "transformer.opm.dev/networking@v0"
				description: "Exposes components via Kubernetes Services"
			}
			requiredResources: {}
			optionalResources: {}
			requiredTraits: {
				"opm.dev/traits/networking@v0#Expose": _
			}
			optionalTraits: {}

			#transform: {
				#component: _
				context:    core.#TransformerContext

				let _ports = [
					for _, p in #component.spec.expose.ports {
						port:       *p.exposedPort | p.targetPort
						targetPort: p.targetPort
						protocol:   *p.protocol | "TCP"
					},
				]

				output: {
					apiVersion: "v1"
					kind:       "Service"
					metadata: {
						name:      #component.metadata.name
						namespace: context.namespace
						labels:    context.labels
					}
					spec: {
						type: #component.spec.expose.type
						ports: _ports
						selector: "app.kubernetes.io/name": #component.metadata.name
					}
				}
			}
		}
	}
}

// 2. Compute matching plan using core.#Matches
// Note: This is inlined here rather than using core.#MatchTransformers because
// CUE's evaluation order with parameterized function patterns can cause issues
// in some contexts. The logic is identical to #MatchTransformers.
matchingPlan: {
	// Iterate over all transformers in the provider
	for tID, t in provider.transformers {
		// Find all components in the module that match this transformer
		let matches = [
			for _, c in allBlueprintsModuleRelease.components
			if (core.#Matches & {transformer: t, component: c}).result {
				c
			},
		]

		// Only include this transformer if it matched at least one component
		if len(matches) > 0 {
			(tID): {
				transformer: t
				components:  matches
			}
		}
	}
}
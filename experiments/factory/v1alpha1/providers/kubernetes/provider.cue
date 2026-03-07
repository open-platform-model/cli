package kubernetes

import (
	provider "opmodel.dev/core/provider@v1"
	k8s_transformers "opmodel.dev/providers/kubernetes/transformers"
)

// KubernetesProvider transforms OPM components to Kubernetes native resources
#Provider: provider.#Provider & {
	metadata: {
		name:        "kubernetes"
		description: "Transforms OPM components to Kubernetes native resources"
		version:     "0.1.0"

		labels: {
			"core.opmodel.dev/format":   "kubernetes"
			"core.opmodel.dev/platform": "container-orchestrator"
		}
	}

	// Transformer registry - maps transformer FQNs to transformer definitions
	#transformers: {
		(k8s_transformers.#DeploymentTransformer.metadata.fqn):             k8s_transformers.#DeploymentTransformer
		(k8s_transformers.#StatefulsetTransformer.metadata.fqn):            k8s_transformers.#StatefulsetTransformer
		(k8s_transformers.#DaemonSetTransformer.metadata.fqn):              k8s_transformers.#DaemonSetTransformer
		(k8s_transformers.#JobTransformer.metadata.fqn):                    k8s_transformers.#JobTransformer
		(k8s_transformers.#CronJobTransformer.metadata.fqn):                k8s_transformers.#CronJobTransformer
		(k8s_transformers.#ServiceTransformer.metadata.fqn):                k8s_transformers.#ServiceTransformer
		(k8s_transformers.#PVCTransformer.metadata.fqn):                    k8s_transformers.#PVCTransformer
		(k8s_transformers.#ConfigMapTransformer.metadata.fqn):              k8s_transformers.#ConfigMapTransformer
		(k8s_transformers.#SecretTransformer.metadata.fqn):                 k8s_transformers.#SecretTransformer
		(k8s_transformers.#ServiceAccountTransformer.metadata.fqn):         k8s_transformers.#ServiceAccountTransformer
		(k8s_transformers.#HPATransformer.metadata.fqn):                    k8s_transformers.#HPATransformer
		(k8s_transformers.#IngressTransformer.metadata.fqn):                k8s_transformers.#IngressTransformer
		(k8s_transformers.#CRDTransformer.metadata.fqn):                    k8s_transformers.#CRDTransformer
		(k8s_transformers.#ServiceAccountResourceTransformer.metadata.fqn): k8s_transformers.#ServiceAccountResourceTransformer
		(k8s_transformers.#RoleTransformer.metadata.fqn):                   k8s_transformers.#RoleTransformer
	}
}

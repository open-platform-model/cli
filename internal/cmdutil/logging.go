package cmdutil

import "github.com/opmodel/cli/internal/output"

// LogResolvedKubernetesConfig emits the resolved Kubernetes config at debug level.
func LogResolvedKubernetesConfig(k8sConfigNamespace, kubeconfig, contextName string) {
	output.Debug("resolved kubernetes config",
		"kubeconfig", kubeconfig,
		"context", contextName,
		"namespace", k8sConfigNamespace,
	)
}

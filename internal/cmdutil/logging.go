package cmdutil

import "github.com/open-platform-model/cli/internal/output"

// defaultKubeconfigDiscovery names client-go's discovery chain, shown when no
// kubeconfig path is configured.
const defaultKubeconfigDiscovery = "default (KUBECONFIG, ~/.kube/config, in-cluster)"

// LogResolvedKubernetesConfig emits the resolved Kubernetes config at debug level.
// An empty kubeconfig is the unconfigured default, logged as its discovery chain.
func LogResolvedKubernetesConfig(k8sConfigNamespace, kubeconfig, contextName string) {
	if kubeconfig == "" {
		kubeconfig = defaultKubeconfigDiscovery
	}
	output.Debug("resolved kubernetes config",
		"kubeconfig", kubeconfig,
		"context", contextName,
		"namespace", k8sConfigNamespace,
	)
}

// Package cmd provides CLI command implementations.
package cmd

import (
	"github.com/opmodel/cli/internal/config"
)

// resolveCommandKubernetes resolves Kubernetes configuration values for a command.
// It uses the command's local flag values and resolves them against env/config/defaults.
func resolveCommandKubernetes(kubeconfigFlag, contextFlag, namespaceFlag, providerFlag string) (*config.ResolvedKubernetesConfig, error) {
	opmConfig := GetOPMConfig()

	var cfg *config.Config
	var providerNames []string

	if opmConfig != nil {
		cfg = opmConfig.Config
		if opmConfig.Providers != nil {
			for name := range opmConfig.Providers {
				providerNames = append(providerNames, name)
			}
		}
	}

	return config.ResolveKubernetes(config.ResolveKubernetesOptions{
		KubeconfigFlag: kubeconfigFlag,
		ContextFlag:    contextFlag,
		NamespaceFlag:  namespaceFlag,
		ProviderFlag:   providerFlag,
		Config:         cfg,
		ProviderNames:  providerNames,
	})
}

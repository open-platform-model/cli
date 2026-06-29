package cmdutil

import (
	"fmt"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/opmodel/cli/internal/config"
)

// ResolvedInstanceTarget bundles selector and Kubernetes config for instance commands.
// Was: ResolvedReleaseTarget (enhancement 0002 D10). The Selector type
// (ReleaseSelectorFlags) is renamed in the X4 slice.
type ResolvedInstanceTarget struct {
	Selector  *ReleaseSelectorFlags
	K8sConfig *config.ResolvedKubernetesConfig
	Namespace string
	LogName   string
}

// ResolveInstanceTarget resolves an instance identifier into selector flags and
// Kubernetes config. Was: ResolveReleaseTarget.
func ResolveInstanceTarget(identifier string, cfg *config.GlobalConfig, kf *K8sFlags, namespaceFlag string) (*ResolvedInstanceTarget, error) {
	ra, err := ResolveInstanceArg(identifier, cfg)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}
	rsf := ra.ToSelectorFlags(namespaceFlag)
	if err := rsf.Validate(); err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  ra.EffectiveNamespace(namespaceFlag),
	})
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}
	if err := RequireNamespace(k8sConfig); err != nil {
		return nil, err
	}

	return &ResolvedInstanceTarget{
		Selector:  rsf,
		K8sConfig: k8sConfig,
		Namespace: k8sConfig.Namespace.Value,
		LogName:   rsf.LogName(),
	}, nil
}

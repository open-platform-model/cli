package cmdutil

import (
	"fmt"

	"github.com/opmodel/cli/internal/config"
	oerrors "github.com/opmodel/cli/pkg/errors"
)

// ResolvedReleaseTarget bundles selector and Kubernetes config for release commands.
type ResolvedReleaseTarget struct {
	Selector  *ReleaseSelectorFlags
	K8sConfig *config.ResolvedKubernetesConfig
	Namespace string
	LogName   string
}

// ResolveReleaseTarget resolves a release identifier into selector flags and Kubernetes config.
func ResolveReleaseTarget(identifier string, cfg *config.GlobalConfig, kf *K8sFlags, namespaceFlag string) (*ResolvedReleaseTarget, error) {
	ra, err := ResolveReleaseArg(identifier, cfg)
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}
	rsf := ra.ToSelectorFlags(namespaceFlag)
	if err := rsf.Validate(); err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}

	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
		NamespaceFlag:  ra.EffectiveNamespace(namespaceFlag),
	})
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}
	if err := RequireNamespace(k8sConfig); err != nil {
		return nil, err
	}

	return &ResolvedReleaseTarget{
		Selector:  rsf,
		K8sConfig: k8sConfig,
		Namespace: k8sConfig.Namespace.Value,
		LogName:   rsf.LogName(),
	}, nil
}

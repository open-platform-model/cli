package cmdutil

import (
	"context"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/open-platform-model/cli/internal/config"
)

// ResolvedInstanceTarget bundles selector and Kubernetes config for instance commands.
type ResolvedInstanceTarget struct {
	Selector  *InstanceSelectorFlags
	K8sConfig *config.ResolvedKubernetesConfig
	Namespace string
	LogName   string
}

// ResolveInstanceTarget resolves an instance identifier into selector flags and
// Kubernetes config. A path identifier is acquired through the kernel, so ctx
// is the command's context. Was: ResolveReleaseTarget.
func ResolveInstanceTarget(ctx context.Context, identifier string, cfg *config.GlobalConfig, kf *K8sFlags, namespaceFlag string) (*ResolvedInstanceTarget, error) {
	ra, err := ResolveInstanceArg(ctx, identifier, cfg)
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

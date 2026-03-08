package cmdutil

import (
	"fmt"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/kubernetes"
	oerrors "github.com/opmodel/cli/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// NewK8sClient creates a Kubernetes client from pre-resolved Kubernetes configuration.
// All values in k8sConfig must already be resolved via config.ResolveKubernetes —
// no further precedence resolution is performed here or inside the client.
// Returns an *ExitError with ExitConnectivityError on failure.
func NewK8sClient(k8sConfig *config.ResolvedKubernetesConfig, apiWarnings string) (*kubernetes.Client, error) {
	client, err := kubernetes.NewClient(kubernetes.ClientOptions{
		Kubeconfig:  k8sConfig.Kubeconfig.Value,
		Context:     k8sConfig.Context.Value,
		APIWarnings: apiWarnings,
	})
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitConnectivityError, Err: err}
	}
	return client, nil
}

// RequireNamespace returns an error if the resolved namespace is empty (i.e.
// no namespace was provided via flag, environment variable, or config file).
// Call this in commands that cannot derive their namespace from a release
// definition (list, status, tree, events, delete).
func RequireNamespace(k8sConfig *config.ResolvedKubernetesConfig) error {
	if k8sConfig.Namespace.Value == "" {
		return &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("namespace is required: use -n flag, OPM_NAMESPACE env var, or set kubernetes.namespace in ~/.opm/config.cue"),
		}
	}
	return nil
}

// ExitCodeFromK8sError maps Kubernetes API errors to exit codes.
func ExitCodeFromK8sError(err error) int {
	switch {
	case apierrors.IsNotFound(err):
		return oerrors.ExitNotFound
	case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
		return oerrors.ExitPermissionDenied
	case apierrors.IsServerTimeout(err), apierrors.IsServiceUnavailable(err):
		return oerrors.ExitConnectivityError
	default:
		return oerrors.ExitGeneralError
	}
}

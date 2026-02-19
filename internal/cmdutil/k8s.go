package cmdutil

import (
	"github.com/opmodel/cli/internal/config"
	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/kubernetes"
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

package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd"
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
		return nil, &opmexit.ExitError{Code: opmexit.ExitConnectivityError, Err: err}
	}
	return client, nil
}

// RequireNamespace returns an error if the resolved namespace is empty (i.e.
// no namespace was provided via flag, environment variable, or config file).
// Call this in commands that cannot derive their namespace from an instance
// definition (list, status, tree, events, delete).
func RequireNamespace(k8sConfig *config.ResolvedKubernetesConfig) error {
	if k8sConfig.Namespace.Value == "" {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("namespace is required: use -n flag, OPM_NAMESPACE env var, or set kubernetes.namespace in ~/.opm/config.cue"),
		}
	}
	return nil
}

// ExitCodeFromK8sError maps an error from a Kubernetes request to the exit code
// of its cause. It reads only the typed error chain, so a %w-wrapped error keeps
// its code: NotFound is ExitNotFound; Forbidden and Unauthorized are
// ExitPermissionDenied; ServerTimeout, ServiceUnavailable and a request with no
// response from the cluster (see noResponse) are ExitConnectivityError; any
// other error, a cancellation among them, is ExitGeneralError.
func ExitCodeFromK8sError(err error) int {
	switch {
	case apierrors.IsNotFound(err):
		return opmexit.ExitNotFound
	case apierrors.IsForbidden(err), apierrors.IsUnauthorized(err):
		return opmexit.ExitPermissionDenied
	case apierrors.IsServerTimeout(err), apierrors.IsServiceUnavailable(err), noResponse(err):
		return opmexit.ExitConnectivityError
	default:
		return opmexit.ExitGeneralError
	}
}

// noResponse reports whether the request got no answer from the cluster: the
// chain holds a net.Error (a refused connection, a host name that does not
// resolve, a timeout, a failed TLS handshake) or context.DeadlineExceeded. A
// chain that holds context.Canceled is the caller's cancellation, not a
// connectivity failure, also when a *url.Error (a net.Error) wraps it, so it is
// checked first.
func noResponse(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr) || errors.Is(err, context.DeadlineExceeded)
}

// IsNoKubeContext reports whether a client-building error means no kubeconfig
// context resolves at all: the kubeconfig file is empty or no file is found by
// default discovery (clientcmd's empty-config error), or an explicitly named
// file (--kubeconfig, OPM_KUBECONFIG, config) does not exist (a not-exist
// error). Default discovery also falls back to the in-cluster service account,
// so a pod with no kubeconfig file resolves a context and never reaches here.
// Any other failure (a malformed file, an unknown --context) is a broken
// kubeconfig, not an absent one.
func IsNoKubeContext(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	return isEmptyKubeconfig(err)
}

// isEmptyKubeconfig walks err's chain for clientcmd's empty-config error,
// which clientcmd.IsEmptyConfig only recognizes unwrapped.
func isEmptyKubeconfig(err error) bool {
	if err == nil {
		return false
	}
	if clientcmd.IsEmptyConfig(err) {
		return true
	}
	switch u := err.(type) { //nolint:errorlint // walking the chain by hand: IsEmptyConfig asserts types, not chains
	case interface{ Unwrap() error }:
		return isEmptyKubeconfig(u.Unwrap())
	case interface{ Unwrap() []error }:
		for _, e := range u.Unwrap() {
			if isEmptyKubeconfig(e) {
				return true
			}
		}
	}
	return false
}

package cmdutil

import (
	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/kubernetes"
)

// K8sClientOpts holds options for creating a Kubernetes client.
type K8sClientOpts struct {
	Kubeconfig  string
	Context     string
	APIWarnings string
}

// NewK8sClient creates a Kubernetes client or returns an *ExitError
// with ExitConnectivityError on failure.
func NewK8sClient(opts K8sClientOpts) (*kubernetes.Client, error) {
	client, err := kubernetes.NewClient(kubernetes.ClientOptions{
		Kubeconfig:  opts.Kubeconfig,
		Context:     opts.Context,
		APIWarnings: opts.APIWarnings,
	})
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitConnectivityError, Err: err}
	}
	return client, nil
}

package cmdutil

import (
	"github.com/opmodel/cli/internal/kubernetes"
)

// K8s exit codes — mirrors internal/cmd constants.
const (
	exitConnectivityError = 3
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
		return nil, &ExitError{Code: exitConnectivityError, Err: err}
	}
	return client, nil
}

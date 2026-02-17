package cmdutil

import (
	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/kubernetes"
)

// NewK8sClient creates a Kubernetes client or returns an *ExitError
// with ExitConnectivityError on failure.
func NewK8sClient(opts kubernetes.ClientOptions) (*kubernetes.Client, error) {
	client, err := kubernetes.NewClient(opts)
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitConnectivityError, Err: err}
	}
	return client, nil
}

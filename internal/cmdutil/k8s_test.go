package cmdutil

import (
	"errors"
	"testing"

	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewK8sClient_InvalidKubeconfig(t *testing.T) {
	// Using an invalid kubeconfig path should cause a failure
	_, err := NewK8sClient(K8sClientOpts{
		Kubeconfig: "/nonexistent/path/kubeconfig",
		Context:    "nonexistent-context",
	})

	require.Error(t, err)
	var exitErr *oerrors.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, oerrors.ExitConnectivityError, exitErr.Code)
}

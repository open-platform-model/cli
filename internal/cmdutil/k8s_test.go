package cmdutil

import (
	"errors"
	"testing"

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
	var exitErr *ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, exitConnectivityError, exitErr.Code)
}

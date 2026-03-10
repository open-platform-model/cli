package cmdutil

import (
	"errors"
	"testing"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/opmodel/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewK8sClient_InvalidKubeconfig(t *testing.T) {
	// Using an invalid kubeconfig path should cause a failure
	k8sConfig := &config.ResolvedKubernetesConfig{
		Kubeconfig: config.ResolvedField{Value: "/nonexistent/path/kubeconfig", Source: config.SourceFlag},
		Context:    config.ResolvedField{Value: "nonexistent-context", Source: config.SourceFlag},
	}
	_, err := NewK8sClient(k8sConfig, "")

	require.Error(t, err)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitConnectivityError, exitErr.Code)
}

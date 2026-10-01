package cmdutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/kubernetes"
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

// With no configured kubeconfig the resolved path is empty, client-go's
// default discovery runs, and a discovery that finds no file (here KUBECONFIG
// names a missing one, so the home file is not consulted either) is still a
// "no kube context" case. In-cluster config is the discovery's last step and
// needs a service account token, so it cannot be exercised here.
func TestNewK8sClient_DefaultDiscoveryNoKubeconfigIsNoContext(t *testing.T) {
	kubernetes.ResetClient()
	t.Cleanup(kubernetes.ResetClient)
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "missing"))

	k8sConfig := &config.ResolvedKubernetesConfig{
		Kubeconfig: config.ResolvedField{Source: config.SourceDefault},
	}
	_, err := NewK8sClient(k8sConfig, "")

	require.Error(t, err)
	assert.True(t, IsNoKubeContext(err), "no kubeconfig anywhere is the no-context case: %v", err)
}

func TestNewK8sClient_DefaultDiscoveryHonoursKUBECONFIG(t *testing.T) {
	kubernetes.ResetClient()
	t.Cleanup(kubernetes.ResetClient)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(path, []byte(`apiVersion: v1
kind: Config
clusters:
- name: c
  cluster:
    server: https://from-env.example:6443
contexts:
- name: ctx
  context:
    cluster: c
    user: u
current-context: ctx
users:
- name: u
  user:
    token: t
`), 0o600))
	t.Setenv("KUBECONFIG", path)

	client, err := NewK8sClient(&config.ResolvedKubernetesConfig{
		Kubeconfig: config.ResolvedField{Source: config.SourceDefault},
	}, "")
	require.NoError(t, err)
	assert.Equal(t, "https://from-env.example:6443", client.RestConfig.Host)
}

// A path named explicitly that does not exist stays the no-context case, as
// before the default became client-go discovery.
func TestIsNoKubeContext_ExplicitMissingPath(t *testing.T) {
	kubernetes.ResetClient()
	t.Cleanup(kubernetes.ResetClient)

	_, err := NewK8sClient(&config.ResolvedKubernetesConfig{
		Kubeconfig: config.ResolvedField{Value: filepath.Join(t.TempDir(), "missing"), Source: config.SourceFlag},
	}, "")
	require.Error(t, err)
	assert.True(t, IsNoKubeContext(err))
}

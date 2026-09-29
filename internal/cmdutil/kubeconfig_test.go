package cmdutil

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// explicitClientConfig loads a kubeconfig the way the CLI does: the resolved
// kubeconfig path (flag, env, config or the ~/.kube/config default) is always
// passed as the explicit path, never through client-go's KUBECONFIG discovery.
func explicitClientConfig(path string) clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: path},
		&clientcmd.ConfigOverrides{},
	)
}

// TestKubeconfig_NoContextShapes pins what client-go returns for the two
// "no kubeconfig context" cases the CLI meets: an empty kubeconfig file is
// clientcmd's empty-config error, and a kubeconfig path that does not exist
// is a not-exist error, not the empty-config one, because the CLI always
// passes the resolved path explicitly.
func TestKubeconfig_NoContextShapes(t *testing.T) {
	dir := t.TempDir()

	empty := filepath.Join(dir, "empty")
	require.NoError(t, os.WriteFile(empty, nil, 0o600))
	_, err := explicitClientConfig(empty).ClientConfig()
	require.Error(t, err)
	assert.True(t, clientcmd.IsEmptyConfig(err), "an empty kubeconfig file is the empty-config case: %v", err)

	_, err = explicitClientConfig(filepath.Join(dir, "missing")).ClientConfig()
	require.Error(t, err)
	assert.False(t, clientcmd.IsEmptyConfig(err), "a missing explicit kubeconfig is not the empty-config case: %v", err)
	assert.True(t, errors.Is(err, os.ErrNotExist), "a missing explicit kubeconfig is a not-exist error: %v", err)
}

// TestKubeconfig_UnroutableServerHonoursDeadline pins that a read against an
// API server that never answers returns once the caller's context deadline
// passes, so a bounded lookup cannot hang a command.
func TestKubeconfig_UnroutableServerHonoursDeadline(t *testing.T) {
	// 192.0.2.0/24 is TEST-NET-1 (RFC 5737): never routed.
	dyn, err := dynamic.NewForConfig(&rest.Config{Host: "https://192.0.2.1:6443"})
	require.NoError(t, err)

	const bound = 500 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), bound)
	defer cancel()

	start := time.Now()
	gvr := schema.GroupVersionResource{Group: "opmodel.dev", Version: "v1alpha1", Resource: "platforms"}
	_, err = dyn.Resource(gvr).Get(ctx, "cluster", metav1.GetOptions{})
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Less(t, elapsed, bound+5*time.Second, "the read returns shortly after the deadline, not after a TCP timeout")
}

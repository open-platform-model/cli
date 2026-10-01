package kubernetes

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEnsureNamespace(t *testing.T) {
	ctx := context.Background()

	t.Run("creates namespace when missing", func(t *testing.T) {
		fakeClientset := fake.NewSimpleClientset()
		client := &Client{Clientset: fakeClientset}

		created, err := client.EnsureNamespace(ctx, "my-ns", false)
		assert.NoError(t, err)
		assert.True(t, created)

		// Verify namespace was created
		ns, err := fakeClientset.CoreV1().Namespaces().Get(ctx, "my-ns", metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, "my-ns", ns.Name)
	})

	t.Run("returns false when namespace exists", func(t *testing.T) {
		fakeClientset := fake.NewSimpleClientset(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "existing-ns"},
		})
		client := &Client{Clientset: fakeClientset}

		created, err := client.EnsureNamespace(ctx, "existing-ns", false)
		assert.NoError(t, err)
		assert.False(t, created)
	})

	t.Run("dry run does not create namespace", func(t *testing.T) {
		fakeClientset := fake.NewSimpleClientset()
		client := &Client{Clientset: fakeClientset}

		created, err := client.EnsureNamespace(ctx, "my-ns", true)
		assert.NoError(t, err)
		assert.True(t, created)

		// Verify namespace was NOT actually created
		_, err = fakeClientset.CoreV1().Namespaces().Get(ctx, "my-ns", metav1.GetOptions{})
		assert.Error(t, err, "namespace should not exist after dry run")
	})
}

func TestBuildRestConfig_InvalidPath(t *testing.T) {
	// buildRestConfig with a nonexistent kubeconfig path should return an error.
	// Values are treated as pre-resolved — no further env/precedence resolution occurs.
	_, err := buildRestConfig(ClientOptions{
		Kubeconfig: "/nonexistent/path/kubeconfig",
		Context:    "nonexistent-context",
	})
	assert.Error(t, err, "expected error for nonexistent kubeconfig path")
}

// writeKubeconfig writes a one-context kubeconfig whose server is server.
func writeKubeconfig(t *testing.T, server string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(path, []byte(`apiVersion: v1
kind: Config
clusters:
- name: c
  cluster:
    server: `+server+`
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
	return path
}

// An empty Kubeconfig defers to client-go's default discovery, which honors
// the KUBECONFIG env var (and, failing that, ~/.kube/config, then in-cluster).
func TestBuildRestConfig_EmptyPathHonoursKUBECONFIG(t *testing.T) {
	t.Setenv("KUBECONFIG", writeKubeconfig(t, "https://from-env.example:6443"))

	cfg, err := buildRestConfig(ClientOptions{})
	require.NoError(t, err)
	assert.Equal(t, "https://from-env.example:6443", cfg.Host)
}

// An explicit path (flag, OPM_KUBECONFIG or config) wins over KUBECONFIG.
func TestBuildRestConfig_ExplicitPathBeatsKUBECONFIG(t *testing.T) {
	t.Setenv("KUBECONFIG", writeKubeconfig(t, "https://from-env.example:6443"))
	explicit := writeKubeconfig(t, "https://explicit.example:6443")

	cfg, err := buildRestConfig(ClientOptions{Kubeconfig: explicit})
	require.NoError(t, err)
	assert.Equal(t, "https://explicit.example:6443", cfg.Host)
}

// --context still selects among the contexts of the discovered kubeconfig.
func TestBuildRestConfig_ContextWithDiscovery(t *testing.T) {
	t.Setenv("KUBECONFIG", writeKubeconfig(t, "https://from-env.example:6443"))

	_, err := buildRestConfig(ClientOptions{Context: "no-such-context"})
	require.Error(t, err)

	cfg, err := buildRestConfig(ClientOptions{Context: "ctx"})
	require.NoError(t, err)
	assert.Equal(t, "https://from-env.example:6443", cfg.Host)
}

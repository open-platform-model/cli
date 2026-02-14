package kubernetes

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestExpandTilde(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err, "should get home directory")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no tilde",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "tilde only",
			input:    "~",
			expected: homeDir,
		},
		{
			name:     "tilde with slash - kubeconfig case",
			input:    "~/.kube/config",
			expected: filepath.Join(homeDir, ".kube", "config"),
		},
		{
			name:     "tilde with path",
			input:    "~/Documents/kubeconfig.yaml",
			expected: filepath.Join(homeDir, "Documents", "kubeconfig.yaml"),
		},
		{
			name:     "tilde username pattern (not expanded)",
			input:    "~otheruser/config",
			expected: "~otheruser/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandTilde(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

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

func TestResolveKubeconfigExpandsTilde(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err, "should get home directory")

	tests := []struct {
		name          string
		flagValue     string
		opmKubeconfig string
		kubeconfig    string
		expected      string
	}{
		{
			name:      "flag with tilde",
			flagValue: "~/.kube/custom-config",
			expected:  filepath.Join(homeDir, ".kube", "custom-config"),
		},
		{
			name:          "OPM_KUBECONFIG env with tilde",
			flagValue:     "",
			opmKubeconfig: "~/.kube/opm-config",
			expected:      filepath.Join(homeDir, ".kube", "opm-config"),
		},
		{
			name:       "KUBECONFIG env with tilde",
			flagValue:  "",
			kubeconfig: "~/configs/k8s.yaml",
			expected:   filepath.Join(homeDir, "configs", "k8s.yaml"),
		},
		{
			name:      "flag with absolute path (no tilde)",
			flagValue: "/etc/kubernetes/admin.conf",
			expected:  "/etc/kubernetes/admin.conf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env vars
			origOPM := os.Getenv("OPM_KUBECONFIG")
			origKube := os.Getenv("KUBECONFIG")
			defer func() {
				os.Setenv("OPM_KUBECONFIG", origOPM)
				os.Setenv("KUBECONFIG", origKube)
			}()

			// Set test env vars
			if tt.opmKubeconfig != "" {
				os.Setenv("OPM_KUBECONFIG", tt.opmKubeconfig)
			} else {
				os.Unsetenv("OPM_KUBECONFIG")
			}

			if tt.kubeconfig != "" {
				os.Setenv("KUBECONFIG", tt.kubeconfig)
			} else {
				os.Unsetenv("KUBECONFIG")
			}

			result := resolveKubeconfig(tt.flagValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

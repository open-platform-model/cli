package kubernetes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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

package kubernetes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveKubeconfig(t *testing.T) {
	tests := []struct {
		name       string
		flagValue  string
		envOPM     string
		envKube    string
		wantFlag   bool // if true, expect flagValue
		wantEnvOPM bool // if true, expect OPM_KUBECONFIG
		wantEnvK   bool // if true, expect KUBECONFIG
		wantHome   bool // if true, expect ~/.kube/config
	}{
		{
			name:      "flag takes precedence",
			flagValue: "/custom/kubeconfig",
			envOPM:    "/opm/kubeconfig",
			envKube:   "/env/kubeconfig",
			wantFlag:  true,
		},
		{
			name:       "OPM_KUBECONFIG takes precedence over KUBECONFIG",
			flagValue:  "",
			envOPM:     "/opm/kubeconfig",
			envKube:    "/env/kubeconfig",
			wantEnvOPM: true,
		},
		{
			name:     "KUBECONFIG used when no flag or OPM env",
			envKube:  "/env/kubeconfig",
			wantEnvK: true,
		},
		{
			name:     "falls back to ~/.kube/config",
			wantHome: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env
			t.Setenv("OPM_KUBECONFIG", tt.envOPM)
			t.Setenv("KUBECONFIG", tt.envKube)

			result := resolveKubeconfig(tt.flagValue)

			switch {
			case tt.wantFlag:
				assert.Equal(t, tt.flagValue, result)
			case tt.wantEnvOPM:
				assert.Equal(t, tt.envOPM, result)
			case tt.wantEnvK:
				assert.Equal(t, tt.envKube, result)
			case tt.wantHome:
				home, err := os.UserHomeDir()
				require.NoError(t, err)
				assert.Equal(t, filepath.Join(home, ".kube", "config"), result)
			}
		})
	}
}

func TestResetClient(t *testing.T) {
	// Ensure reset clears cached client
	clientMu.Lock()
	cachedClient = &Client{} // set a dummy
	clientMu.Unlock()

	ResetClient()

	clientMu.Lock()
	defer clientMu.Unlock()
	assert.Nil(t, cachedClient)
}

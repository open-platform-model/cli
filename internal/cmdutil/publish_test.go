package cmdutil

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/publish"
)

// TestPublishError_ExitCodes pins the pipeline-error → exit-code mapping:
// a failed registry operation is 3, anything else unexpected is 1.
func TestPublishError_ExitCodes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "connectivity maps to ExitConnectivityError",
			err:  &publish.ConnectivityError{Op: "listing published versions", Err: errors.New("dial tcp: refused")},
			want: opmexit.ExitConnectivityError,
		},
		{
			name: "wrapped connectivity still maps to ExitConnectivityError",
			err:  fmt.Errorf("publishing: %w", &publish.ConnectivityError{Op: "push", Err: errors.New("timeout")}),
			want: opmexit.ExitConnectivityError,
		},
		{
			// The compatibility walk's mid-flight abort is the same error
			// class as the lookup's: the artifact was never judged, exit 3.
			name: "compat-walk abort maps to ExitConnectivityError",
			err:  &publish.ConnectivityError{Op: "loading example.com/catalogs/demo/resources/v1beta1@v1.0.0", Err: errors.New("dial tcp: refused")},
			want: opmexit.ExitConnectivityError,
		},
		{
			name: "a registry answer maps to ExitConnectivityError",
			err:  &publish.RegistryError{Op: "listing published versions", Err: errors.New("503 Service Unavailable")},
			want: opmexit.ExitConnectivityError,
		},
		{
			name: "a refused credential maps to ExitConnectivityError",
			err:  &publish.RegistryError{Op: "push", Unauthorized: true, Err: errors.New("401 Unauthorized")},
			want: opmexit.ExitConnectivityError,
		},
		{
			name: "anything else maps to ExitGeneralError",
			err:  errors.New("zipping failed"),
			want: opmexit.ExitGeneralError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := publishError(tt.err)
			var exitErr *opmexit.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, tt.want, exitErr.Code)
		})
	}
}

// TestPublishError_LoginHint: only a refused credential points to the login
// command, and the registry's own answer stays in the message.
func TestPublishError_LoginHint(t *testing.T) {
	const hint = "opm registry login"
	cause := errors.New("401 Unauthorized: unauthorized: authentication required")

	refused := publishError(&publish.RegistryError{Op: "pushing example.com/modules/demo:v1.2.0", Unauthorized: true, Err: cause})
	assert.Contains(t, refused.Error(), "registry refused the credentials (authentication or permission)")
	assert.Contains(t, refused.Error(), "pushing example.com/modules/demo:v1.2.0")
	assert.Contains(t, refused.Error(), cause.Error())
	assert.Contains(t, refused.Error(), hint)
	assert.NotContains(t, refused.Error(), "unreachable")
	assert.ErrorIs(t, refused, cause, "the cause stays wrapped")

	for _, err := range []error{
		&publish.RegistryError{Op: "push", Err: errors.New("503 Service Unavailable")},
		&publish.ConnectivityError{Op: "push", Err: errors.New("dial tcp: refused")},
		errors.New("zipping failed"),
	} {
		assert.NotContains(t, publishError(err).Error(), hint, "%v", err)
	}
}

// runPublish runs the shared publish body on an empty directory: every case
// below fails at the core schema fetch, before the directory is read.
func runPublish(t *testing.T, cfg *config.GlobalConfig) error {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	return RunPublish(cmd, cfg, publish.KindModule, []string{t.TempDir()}, &PublishFlags{DryRun: true})
}

// With no registry configured anywhere, a core schema that cannot be loaded
// is reported as the missing configuration (exit 2, opm config init), not as
// an unreachable registry.
func TestRunPublish_NoRegistryConfigured(t *testing.T) {
	// CUE_CACHE_DIR names a regular file, so the fetch fails before it asks
	// any registry and the test needs no network.
	notADir := filepath.Join(t.TempDir(), "cache")
	require.NoError(t, os.WriteFile(notADir, nil, 0o600))
	t.Setenv("CUE_CACHE_DIR", notADir)
	t.Setenv("CUE_REGISTRY", "")

	err := runPublish(t, &config.GlobalConfig{})
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code, "%v", err)
	assert.Contains(t, err.Error(), "no registry is configured: loading core schema: ")
	assert.Contains(t, err.Error(), "opm config init")
	assert.NotContains(t, err.Error(), "unreachable")
}

// The core schema fetch is classified like the lookup and the push: only no
// response is unreachable, and a refused credential points to the login.
func TestRunPublish_SchemaFetchFailureIsNamed(t *testing.T) {
	for _, tc := range []struct {
		name     string
		registry func(t *testing.T) string
		want     string
		login    bool
	}{
		{"refused connection", func(*testing.T) string { return cuemodtest.UnreachableRegistry }, "registry unreachable: loading core schema: ", false},
		{"401", func(t *testing.T) string { return cuemodtest.StatusRegistry(t, http.StatusUnauthorized) }, "registry refused the credentials (authentication or permission): loading core schema: ", true},
		{"503", func(t *testing.T) string { return cuemodtest.StatusRegistry(t, http.StatusServiceUnavailable) }, "registry operation failed: loading core schema: ", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cuemodtest.ColdCache(t)
			err := runPublish(t, &config.GlobalConfig{Registry: tc.registry(t)})
			var exitErr *opmexit.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, opmexit.ExitConnectivityError, exitErr.Code, "%v", err)
			assert.Contains(t, err.Error(), tc.want)
			assert.Equal(t, tc.login, bytes.Contains([]byte(err.Error()), []byte(registryLoginHint)), "%v", err)
		})
	}
}

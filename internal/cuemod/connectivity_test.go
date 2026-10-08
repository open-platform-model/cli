package cuemod

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modregistry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
)

// TestIsConnectivityError_UnreachableRegistry shows a tidy on a cold cache
// with the registry refusing connections is a connectivity failure.
func TestIsConnectivityError_UnreachableRegistry(t *testing.T) {
	cuemodtest.ColdCache(t)
	dir := cuemodtest.WriteConsumer(t, t.TempDir(), cuemodtest.UntidyModuleCue)

	_, err := Tidy(context.Background(), dir, TidyOptions{Registry: cuemodtest.UnreachableRegistry})
	require.Error(t, err)
	assert.True(t, IsConnectivityError(err), "%v", err)
}

// TestIsConnectivityError_RegistryAnswered shows a module the reachable
// registry does not hold is not a connectivity failure.
func TestIsConnectivityError_RegistryAnswered(t *testing.T) {
	cuemodtest.ColdCache(t)
	registry := cuemodtest.Registry(t)
	dir := cuemodtest.WriteConsumer(t, t.TempDir(), cuemodtest.UntidyModuleCue)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "consumer.cue"), []byte(`package consumer

import "example.com/missing@v0"

v: missing.version
`), 0o600))

	_, err := Tidy(context.Background(), dir, TidyOptions{Registry: registry})
	require.Error(t, err)
	assert.False(t, IsConnectivityError(err), "unexpected connectivity classification: %q", err.Error())
}

// TestIsVersionNotHeld_OnlyNotFound holds the kind boundary: a fetch failure
// with no HTTP status that is not a not-found (an archive that does not
// unzip, say) is not "not held", and neither is an unreachable registry.
func TestIsVersionNotHeld_OnlyNotFound(t *testing.T) {
	assert.True(t, IsVersionNotHeld(errors.New("cannot fetch example.com/x@v1.0.0: module not found")))
	assert.False(t, IsVersionNotHeld(errors.New("cannot fetch example.com/x@v1.0.0: zip: not a valid zip file")))
	assert.False(t, IsVersionNotHeld(errors.New("cannot fetch example.com/x@v1.0.0: cannot do HTTP request: connection refused")))
}

// TestIsUnauthorized holds the kind boundary on a real registry client: only
// a 401 or 403 answer is unauthorized; no response, another answer and nil
// are not.
func TestIsUnauthorized(t *testing.T) {
	versions := func(t *testing.T, registry string) error {
		t.Helper()
		resolver, err := modconfig.NewResolver(&modconfig.Config{CUERegistry: registry})
		require.NoError(t, err)
		_, err = modregistry.NewClientWithResolver(resolver).ModuleVersions(context.Background(), cuemodtest.DepModule)
		return err
	}
	for _, tc := range []struct {
		name     string
		registry func(t *testing.T) string
		want     bool
	}{
		{"401", func(t *testing.T) string { return cuemodtest.StatusRegistry(t, http.StatusUnauthorized) }, true},
		{"429", func(t *testing.T) string { return cuemodtest.StatusRegistry(t, http.StatusTooManyRequests) }, false},
		{"503", func(t *testing.T) string { return cuemodtest.StatusRegistry(t, http.StatusServiceUnavailable) }, false},
		{"refused connection", func(*testing.T) string { return cuemodtest.UnreachableRegistry }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := versions(t, tc.registry(t))
			require.Error(t, err)
			assert.Equal(t, tc.want, IsUnauthorized(err), "%v", err)
			if tc.want {
				assert.False(t, IsConnectivityError(err), "a registry that answered was reached")
			}
		})
	}
	assert.False(t, IsUnauthorized(nil))
	// A push reports the registry's answer as flattened text.
	assert.True(t, IsUnauthorized(errors.New("cannot make scratch config: 403 Forbidden: denied: Forbidden")))
}

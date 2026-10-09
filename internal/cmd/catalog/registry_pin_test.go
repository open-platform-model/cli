package catalogcmd

import (
	"context"
	"net/http"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/publish"
)

// TestRegistryCheck_ExitCodes_Pinned pins `opm catalog registry check`'s exit
// code for each way the named build's fetch fails: 5 when the registry
// answers that it does not hold the version (a 403 tag lookup reads the
// same), 3 for every other registry answer or no response, 1 for a failure
// that asks no registry.
func TestRegistryCheck_ExitCodes_Pinned(t *testing.T) {
	held := "example.com/dep@" + cuemodtest.DepNewest
	status := func(code int) func(t *testing.T) string {
		return func(t *testing.T) string { return cuemodtest.StatusRegistry(t, code) }
	}
	for _, tc := range []struct {
		name       string
		registry   func(t *testing.T) string
		coordinate string
		want       int
	}{
		{"version not held", cuemodtest.Registry, "example.com/dep@v0.9.0", opmexit.ExitNotFound},
		{"403 on every request", status(http.StatusForbidden), held, opmexit.ExitNotFound},
		{"401", status(http.StatusUnauthorized), held, opmexit.ExitConnectivityError},
		{"token endpoint answers 401", func(t *testing.T) string { return cuemodtest.TokenRegistry(t, http.StatusUnauthorized) }, held, opmexit.ExitConnectivityError},
		{"token endpoint answers 403", func(t *testing.T) string { return cuemodtest.TokenRegistry(t, http.StatusForbidden) }, held, opmexit.ExitNotFound},
		{"429", status(http.StatusTooManyRequests), held, opmexit.ExitConnectivityError},
		{"503", status(http.StatusServiceUnavailable), held, opmexit.ExitConnectivityError},
		{"refused connection", func(*testing.T) string { return cuemodtest.UnreachableRegistry }, held, opmexit.ExitConnectivityError},
		{"tag held, archive blob 404", func(t *testing.T) string {
			return cuemodtest.Fronted(t, cuemodtest.Registry(t), cuemodtest.BlobAnswers("example.com/dep", http.StatusNotFound))
		}, held, opmexit.ExitConnectivityError},
		{"registry mapping does not parse", func(*testing.T) string { return "::nonsense::" }, held, opmexit.ExitGeneralError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOCKER_CONFIG", t.TempDir())
			cuemodtest.ColdCache(t)
			_, err := publish.RegistryCheck(context.Background(), publish.CheckOptions{
				Coordinate: tc.coordinate,
				Context:    cuecontext.New(),
				Registry:   tc.registry(t),
			})
			require.Error(t, err)
			assert.Equal(t, tc.want, exitCode(t, checkError(err)), "%v", err)
		})
	}
}

func exitCode(t *testing.T, err error) int {
	t.Helper()
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	return exitErr.Code
}

package publish

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
)

// runAgainst runs the module pipeline on a clean tree against registry.
func runAgainst(t *testing.T, registry string) (*Plan, Options, error) {
	t.Helper()
	cueCtx := cuecontext.New()
	opts := Options{
		Dir:            writeTree(t, moduleFiles()),
		Kind:           KindModule,
		Context:        cueCtx,
		Kernel:         kernel.New(kernel.WithRegistry(registry)),
		IdentitySchema: stubSchema(t, cueCtx),
		Registry:       registry,
	}
	p, err := Run(context.Background(), opts)
	return p, opts, err
}

// writesAnswer answers status to every request that is not a read, and
// forwards the reads: a registry that lets the caller look but not push.
func writesAnswer(status int) cuemodtest.Answer {
	return func(r *http.Request) int {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			return 0
		}
		return status
	}
}

// A registry that answered the already-published lookup was reached: only no
// response at all is a *ConnectivityError. A 401 is a refused credential.
func TestGate_AlreadyPublished_RegistryAnswerIsNotUnreachable(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status       int
		unauthorized bool
	}{
		{"401", http.StatusUnauthorized, true},
		{"429", http.StatusTooManyRequests, false},
		{"503", http.StatusServiceUnavailable, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, _, err := runAgainst(t, cuemodtest.StatusRegistry(t, tc.status))
			require.Error(t, err)

			var connErr *ConnectivityError
			assert.False(t, errors.As(err, &connErr), "a registry that answered is not unreachable")
			assert.NotContains(t, err.Error(), "unreachable")
			assert.Contains(t, err.Error(), "listing published versions of example.com/modules/demo@v1")
			assert.Contains(t, err.Error(), http.StatusText(tc.status), "the registry's own answer stays visible")

			var regErr *RegistryError
			require.ErrorAs(t, err, &regErr)
			assert.Equal(t, tc.unauthorized, regErr.Unauthorized)

			// The verdict is as incomplete as for an unreachable registry.
			require.NotNil(t, p)
			assert.False(t, p.RegistryChecked)
			assert.Contains(t, p.Render(), "INCOMPLETE")
		})
	}
}

// A push the registry refuses (the caller may read but not write) is an
// authentication or permission failure, not an unreachable registry.
func TestPush_RefusedCredentialIsNotUnreachable(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			registry := cuemodtest.Fronted(t, emptyTestRegistry(t), writesAnswer(status))
			p, opts, err := runAgainst(t, registry)
			require.NoError(t, err)
			require.True(t, p.Go(), refusalHeadlines(p))

			err = Push(context.Background(), opts, p)
			require.Error(t, err)

			var connErr *ConnectivityError
			assert.False(t, errors.As(err, &connErr), "a registry that answered is not unreachable")
			assert.NotContains(t, err.Error(), "unreachable")
			assert.Contains(t, err.Error(), "pushing example.com/modules/demo:v1.2.0")
			assert.Contains(t, err.Error(), http.StatusText(status), "the registry's own answer stays visible")

			var regErr *RegistryError
			require.ErrorAs(t, err, &regErr)
			assert.True(t, regErr.Unauthorized)
		})
	}
}

// No response at all stays a *ConnectivityError on the push too.
func TestPush_UnreachableRegistryIsConnectivity(t *testing.T) {
	p, opts, err := runAgainst(t, emptyTestRegistry(t))
	require.NoError(t, err)
	require.True(t, p.Go(), refusalHeadlines(p))

	// The lookup's client is dropped so the push resolves the unreachable
	// mapping itself.
	p.registryClient = nil
	opts.Registry = cuemodtest.UnreachableRegistry
	err = Push(context.Background(), opts, p)
	var connErr *ConnectivityError
	require.ErrorAs(t, err, &connErr)
	assert.Contains(t, err.Error(), "registry unreachable: pushing example.com/modules/demo:v1.2.0")
}

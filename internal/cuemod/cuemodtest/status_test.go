package cuemodtest

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/require"
)

// fetchStatus fetches DepModule@DepNewest through modconfig, as every cli
// registry read does, and returns the HTTP status the client saw (0 when the
// fetch succeeded or carried no status).
func fetchStatus(t *testing.T, registry string) int {
	t.Helper()
	ColdCache(t)
	reg, err := modconfig.NewRegistry(&modconfig.Config{Env: append(os.Environ(), "CUE_REGISTRY="+registry)})
	require.NoError(t, err)
	mv, err := module.NewVersion(DepModule, DepNewest)
	require.NoError(t, err)
	_, err = reg.Fetch(context.Background(), mv)
	var he ociregistry.HTTPError
	if errors.As(err, &he) {
		return he.StatusCode()
	}
	return 0
}

func TestStatusRegistry_StatusReachesFetch(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		require.Equal(t, status, fetchStatus(t, StatusRegistry(t, status)))
	}
}

func TestFronted_RepoAnswers(t *testing.T) {
	require.Equal(t, http.StatusServiceUnavailable,
		fetchStatus(t, Fronted(t, Registry(t), RepoAnswers("example.com/dep", http.StatusServiceUnavailable))))
	require.Equal(t, 0, fetchStatus(t, Fronted(t, Registry(t), RepoAnswers("example.com/other", http.StatusServiceUnavailable))),
		"a request for another repository is forwarded")
}

func TestFronted_BlobAnswers(t *testing.T) {
	require.Equal(t, http.StatusNotFound,
		fetchStatus(t, Fronted(t, Registry(t), BlobAnswers("example.com/dep", http.StatusNotFound))))
}

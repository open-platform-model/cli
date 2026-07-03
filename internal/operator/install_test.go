package operator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveManifest_EmptyVersionUsesEmbedded(t *testing.T) {
	objs, version, source, err := resolveManifest(context.Background(), "")
	require.NoError(t, err)

	assert.Equal(t, PinnedOperatorVersion, version)
	assert.Equal(t, "embedded", source)
	assert.Len(t, objs, 17)
}

func TestResolveManifest_VersionFetchesInstead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: fetched-ns\n"))
	}))
	defer server.Close()

	objs, version, source, err := resolveManifestFrom(context.Background(), server.URL, "v1.0.0-alpha.3")
	require.NoError(t, err)

	assert.Equal(t, "v1.0.0-alpha.3", version)
	assert.Equal(t, "fetched", source)
	require.Len(t, objs, 1)
	assert.Equal(t, "fetched-ns", objs[0].GetName())
}

func TestResolveManifest_FetchErrorPropagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	objs, version, source, err := resolveManifestFrom(context.Background(), server.URL, "v9.9.9")
	require.Error(t, err)
	assert.Empty(t, objs)
	assert.Empty(t, version)
	assert.Empty(t, source)
	assert.ErrorContains(t, err, "v9.9.9")
}

// Install's apply+wait wiring (--crds-only, --version, --timeout) is
// exercised end-to-end against a real kind cluster in the e2e suite (task
// 6.1) rather than here: the fake dynamic client's Apply reactor doesn't
// support server-side-apply patches against unstructured.Unstructured
// objects (it falls back to reflection-based strategic-merge-patch, which
// only works for typed structs), so it can't stand in for a real apiserver
// on this code path.

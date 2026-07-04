package operator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchManifest_BuildsExpectedURL(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("apiVersion: v1\nkind: Namespace\n"))
	}))
	defer server.Close()

	data, err := fetchManifest(context.Background(), server.URL, "v1.0.0-alpha.3")
	require.NoError(t, err)

	assert.Equal(t, "/v1.0.0-alpha.3/install.yaml", gotPath)
	assert.Contains(t, string(data), "kind: Namespace")
}

func TestFetchManifest_MissingTagErrorsNamingTagAndURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := fetchManifest(context.Background(), server.URL, "v9.9.9")
	require.Error(t, err)
	assert.ErrorContains(t, err, "v9.9.9")
	assert.ErrorContains(t, err, server.URL+"/v9.9.9/install.yaml")
}

func TestFetchManifest_ServerErrorSurfacesStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := fetchManifest(context.Background(), server.URL, "v1.0.0-alpha.2")
	require.Error(t, err)
	assert.ErrorContains(t, err, "500")
}

func TestFetchManifest_UnreachableHostErrors(t *testing.T) {
	_, err := fetchManifest(context.Background(), "https://127.0.0.1:1", "v1.0.0-alpha.2")
	require.Error(t, err)
	assert.ErrorContains(t, err, "v1.0.0-alpha.2")
}

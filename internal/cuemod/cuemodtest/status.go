package cuemodtest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// StatusRegistry starts a registry that answers every request with status
// and returns its CUE_REGISTRY mapping. The answer carries the OCI error
// code a real registry gives that status (see wireCode), so the client sees
// what it would see from a registry that refuses, limits or fails.
func StatusRegistry(t *testing.T, status int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeStatus(w, r, status)
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://") + "+insecure"
}

// Answer picks the status a fronted registry gives one request: 0 forwards
// it to the registry behind.
type Answer func(r *http.Request) int

// RepoAnswers answers status to every request for a repository whose path
// starts with repo (the module path without its major, as CUE names the
// repository), and forwards the rest.
func RepoAnswers(repo string, status int) Answer {
	prefix := "/v2/" + repo + "/"
	return func(r *http.Request) int {
		if strings.HasPrefix(r.URL.Path, prefix) {
			return status
		}
		return 0
	}
}

// BlobAnswers answers status to every blob request for a repository whose
// path starts with repo, and forwards the rest: the registry holds the tag
// and its manifest but not the content.
func BlobAnswers(repo string, status int) Answer {
	prefix := "/v2/" + repo + "/blobs/"
	return func(r *http.Request) int {
		if strings.HasPrefix(r.URL.Path, prefix) {
			return status
		}
		return 0
	}
}

// Fronted starts a registry in front of the one at mapping (a
// CUE_REGISTRY mapping of the form host+insecure) that answers each request
// with the first non-zero status the answers give, and forwards the rest.
// It returns the fronted registry's CUE_REGISTRY mapping.
func Fronted(t *testing.T, mapping string, answers ...Answer) string {
	t.Helper()
	upstream, err := url.Parse("http://" + strings.TrimSuffix(mapping, "+insecure"))
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, answer := range answers {
			if status := answer(r); status != 0 {
				writeStatus(w, r, status)
				return
			}
		}
		proxy.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://") + "+insecure"
}

// writeStatus writes status with the OCI error body a distribution-spec
// registry sends for it.
func writeStatus(w http.ResponseWriter, r *http.Request, status int) {
	code := wireCode(r, status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if code == "" {
		return
	}
	fmt.Fprintf(w, `{"errors":[{"code":%q,"message":%q}]}`, code, http.StatusText(status))
}

// wireCode is the distribution-spec error code for status on request r, or
// "" for a status the spec gives no code (a server error).
func wireCode(r *http.Request, status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "DENIED"
	case http.StatusTooManyRequests:
		return "TOOMANYREQUESTS"
	case http.StatusNotFound:
		switch {
		case strings.Contains(r.URL.Path, "/blobs/"):
			return "BLOB_UNKNOWN"
		case strings.Contains(r.URL.Path, "/manifests/"):
			return "MANIFEST_UNKNOWN"
		default:
			return "NAME_UNKNOWN"
		}
	default:
		return ""
	}
}

// TokenRegistry starts a registry that hands out bearer tokens, as GHCR and
// Docker Hub do: every registry request answers 401 with a Bearer challenge
// naming its own token endpoint, and the token endpoint answers tokenStatus
// (with an empty body, so a 200 carries no token). It returns the registry's
// CUE_REGISTRY mapping.
func TokenRegistry(t *testing.T, tokenStatus int) string {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.WriteHeader(tokenStatus)
			return
		}
		w.Header().Set("Www-Authenticate", fmt.Sprintf(`Bearer realm=%q,service="registry"`, srv.URL+"/token"))
		writeStatus(w, r, http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://") + "+insecure"
}

package cuemod

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/module"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
)

// The answers below are the exit-code decision `opm instance init` takes
// from IsConnectivityError (true exits 3, false exits 1). They were measured
// against the text probe this package used before the library classified
// registry failures, and they hold unchanged across that swap: a registry
// that answered, with any status, is not a connectivity failure.

// tidyErr runs a tidy on an untidy consumer whose import resolves against
// registry, cold cache, and returns the flattened error cmd/cue gives.
func tidyErr(t *testing.T, registry string, missingImport bool) error {
	t.Helper()
	cuemodtest.ColdCache(t)
	dir := cuemodtest.WriteConsumer(t, t.TempDir(), cuemodtest.UntidyModuleCue)
	if missingImport {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "consumer.cue"), []byte(`package consumer

import "example.com/missing@v0"

v: missing.version
`), 0o600))
	}
	_, err := Tidy(context.Background(), dir, TidyOptions{Registry: registry})
	require.Error(t, err)
	return err
}

// acquireErr acquires a module through the kernel, as init does, cold
// cache, and returns the typed error the kernel gives.
func acquireErr(t *testing.T, registry, modPath, version string) error {
	t.Helper()
	cuemodtest.ColdCache(t)
	_, err := kernel.New(kernel.WithRegistry(registry)).AcquireModuleFromRegistry(context.Background(), modPath, version)
	require.Error(t, err)
	return err
}

func TestIsConnectivityError_Pinned(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  func(t *testing.T) error
		want bool
	}{
		{"tidy: refused connection", func(t *testing.T) error { return tidyErr(t, cuemodtest.UnreachableRegistry, false) }, true},
		{"tidy: module not held", func(t *testing.T) error { return tidyErr(t, cuemodtest.Registry(t), true) }, false},
		{"tidy: 403", func(t *testing.T) error { return tidyErr(t, cuemodtest.StatusRegistry(t, http.StatusForbidden), false) }, false},
		{"tidy: 401", func(t *testing.T) error {
			return tidyErr(t, cuemodtest.StatusRegistry(t, http.StatusUnauthorized), false)
		}, false},
		{"tidy: 429", func(t *testing.T) error {
			return tidyErr(t, cuemodtest.StatusRegistry(t, http.StatusTooManyRequests), false)
		}, false},
		{"tidy: 503", func(t *testing.T) error {
			return tidyErr(t, cuemodtest.StatusRegistry(t, http.StatusServiceUnavailable), false)
		}, false},
		{"acquire: refused connection", func(t *testing.T) error {
			return acquireErr(t, cuemodtest.UnreachableRegistry, cuemodtest.DepModule, cuemodtest.DepNewest)
		}, true},
		{"acquire: module not held", func(t *testing.T) error {
			return acquireErr(t, cuemodtest.Registry(t), "example.com/missing@v0", "v0.1.0")
		}, false},
		{"acquire: 403", func(t *testing.T) error {
			return acquireErr(t, cuemodtest.StatusRegistry(t, http.StatusForbidden), cuemodtest.DepModule, cuemodtest.DepNewest)
		}, false},
		{"acquire: 401", func(t *testing.T) error {
			return acquireErr(t, cuemodtest.StatusRegistry(t, http.StatusUnauthorized), cuemodtest.DepModule, cuemodtest.DepNewest)
		}, false},
		{"acquire: 429", func(t *testing.T) error {
			return acquireErr(t, cuemodtest.StatusRegistry(t, http.StatusTooManyRequests), cuemodtest.DepModule, cuemodtest.DepNewest)
		}, false},
		{"acquire: 503", func(t *testing.T) error {
			return acquireErr(t, cuemodtest.StatusRegistry(t, http.StatusServiceUnavailable), cuemodtest.DepModule, cuemodtest.DepNewest)
		}, false},
		{"fetch: expired deadline", func(t *testing.T) error {
			cuemodtest.ColdCache(t)
			reg, err := modconfig.NewRegistry(&modconfig.Config{Env: append(os.Environ(), "CUE_REGISTRY="+cuemodtest.Registry(t))})
			require.NoError(t, err)
			mv, err := module.NewVersion(cuemodtest.DepModule, cuemodtest.DepNewest)
			require.NoError(t, err)
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			_, err = reg.Fetch(ctx, mv)
			require.ErrorIs(t, err, context.DeadlineExceeded)
			return err
		}, true},
		{"nil", func(*testing.T) error { return nil }, false},
		{"canceled request", func(*testing.T) error {
			return fmt.Errorf("fetching: %w", &url.Error{Op: "Get", URL: "http://h/", Err: context.Canceled})
		}, true},
		{"plain cancellation", func(*testing.T) error { return fmt.Errorf("loading: %w", context.Canceled) }, false},
		// context.DeadlineExceeded itself is a net.Error (Timeout reports true).
		{"plain expired deadline", func(*testing.T) error { return fmt.Errorf("loading: %w", context.DeadlineExceeded) }, true},
		{"undeclared import", func(*testing.T) error {
			return errors.New(`cannot find module providing package example.com/missing@v0`)
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.err(t)
			assert.Equal(t, tc.want, IsConnectivityError(err), "%v", err)
		})
	}
}

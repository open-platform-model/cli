package cuemod

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
)

// TestIsConnectivityError_UnreachableRegistry pins the transport-failure
// wording against the embedded CUE version: a tidy on a cold cache with the
// registry refusing connections must classify as a connectivity failure.
func TestIsConnectivityError_UnreachableRegistry(t *testing.T) {
	cuemodtest.ColdCache(t)
	dir := cuemodtest.WriteConsumer(t, t.TempDir(), cuemodtest.UntidyModuleCue)

	_, err := Tidy(context.Background(), dir, TidyOptions{Registry: cuemodtest.UnreachableRegistry})
	require.Error(t, err)
	assert.True(t, IsConnectivityError(err), "cmd/cue's transport-failure wording changed; update IsConnectivityError: %q", err.Error())
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

func TestIsConnectivityError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "wrapped net error", err: fmt.Errorf("fetching: %w", &net.OpError{Op: "dial", Err: errors.New("refused")}), want: true},
		{name: "flattened transport failure", err: errors.New(`cannot fetch x: cannot do HTTP request: Get "http://h/": dial tcp: connection refused`), want: true},
		{name: "resolution failure", err: errors.New(`cannot find module providing package example.com/missing@v0`), want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsConnectivityError(tc.err))
		})
	}
}

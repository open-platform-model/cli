package cuemod

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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

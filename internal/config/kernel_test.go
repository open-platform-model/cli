package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewKernel_ReturnsKernelWithSchemaCache(t *testing.T) {
	for _, registry := range []string{"", "opmodel.dev=ghcr.io/open-platform-model"} {
		k := NewKernel(registry)
		require.NotNil(t, k, "registry %q", registry)
		require.NotNil(t, k.SchemaCache(), "registry %q", registry)
	}
}

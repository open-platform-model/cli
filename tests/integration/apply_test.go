package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/cue"
	"github.com/opmodel/cli/internal/kubernetes"
)

func TestApply_ServerSideApply(t *testing.T) {
	if testClient == nil {
		t.Skip("No test client available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load test module
	loader := cue.NewLoader()
	module, err := loader.LoadModule(ctx, "../fixtures/hello-world", nil)
	require.NoError(t, err)

	// Render manifests
	renderer := cue.NewRenderer()
	manifestSet, err := renderer.RenderModule(ctx, module)
	require.NoError(t, err)
	require.Greater(t, manifestSet.Len(), 0)

	// Apply
	labels := kubernetes.ModuleLabels(
		module.Metadata.Name,
		"default",
		module.Metadata.Version,
		"",
	)

	result, err := testClient.Apply(ctx, manifestSet.Objects(), kubernetes.ApplyOptions{
		Namespace: "default",
		Labels:    labels,
		DryRun:    true, // Use dry-run to avoid actually creating resources
	})

	require.NoError(t, err)
	assert.Greater(t, result.Created, 0)
}

func TestApply_Idempotent(t *testing.T) {
	if testClient == nil {
		t.Skip("No test client available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load test module
	loader := cue.NewLoader()
	module, err := loader.LoadModule(ctx, "../fixtures/hello-world", nil)
	require.NoError(t, err)

	// Render manifests
	renderer := cue.NewRenderer()
	manifestSet, err := renderer.RenderModule(ctx, module)
	require.NoError(t, err)

	labels := kubernetes.ModuleLabels(
		module.Metadata.Name,
		"default",
		module.Metadata.Version,
		"",
	)

	opts := kubernetes.ApplyOptions{
		Namespace: "default",
		Labels:    labels,
		DryRun:    true,
	}

	// Apply twice - should succeed both times with SSA
	result1, err := testClient.Apply(ctx, manifestSet.Objects(), opts)
	require.NoError(t, err)

	result2, err := testClient.Apply(ctx, manifestSet.Objects(), opts)
	require.NoError(t, err)

	// Both should succeed
	assert.Equal(t, result1.Created, result2.Created)
}

package cmdutil

import (
	"testing"
	"time"

	"github.com/opmodel/cli/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildReleaseSummary(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	inv := &inventory.InventorySecret{
		ReleaseMetadata: inventory.ReleaseMetadata{ReleaseName: "demo", ReleaseNamespace: "apps", ReleaseID: "uuid-1", LastTransitionTime: now},
		ModuleMetadata:  inventory.ModuleMetadata{Name: "module-a"},
		Index:           []string{"change-1"},
		Changes: map[string]*inventory.ChangeEntry{
			"change-1": {Source: inventory.ChangeSource{Version: "0.1.0"}, Timestamp: now},
		},
	}

	summary := BuildReleaseSummary(inv)
	assert.Equal(t, "demo", summary.Name)
	assert.Equal(t, "module-a", summary.Module)
	assert.Equal(t, "0.1.0", summary.Version)
	assert.Equal(t, "uuid-1", summary.ReleaseID)
	assert.NotEmpty(t, summary.Age)
}

func TestRenderReleaseListOutput_InvalidJSONPathNotNeeded(t *testing.T) {
	// Smoke test JSON path for compile/runtime coverage.
	err := RenderReleaseListOutput([]ReleaseSummary{{Name: "demo"}}, "json", false)
	require.NoError(t, err)
}

package query

import (
	"testing"
	"time"

	pkginventory "github.com/opmodel/cli/pkg/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildReleaseSummary(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	inv := &pkginventory.InventorySecret{ReleaseMetadata: pkginventory.ReleaseMetadata{ReleaseName: "demo", ReleaseNamespace: "apps", ReleaseID: "uuid-1", LastTransitionTime: now}, ModuleMetadata: pkginventory.ModuleMetadata{Name: "module-a"}, Index: []string{"change-1"}, Changes: map[string]*pkginventory.ChangeEntry{"change-1": {Source: pkginventory.ChangeSource{Version: "0.1.0"}, Timestamp: now}}}
	summary := BuildReleaseSummary(inv)
	assert.Equal(t, "demo", summary.Name)
	assert.Equal(t, "module-a", summary.Module)
	assert.Equal(t, "cli", summary.Owner)
	assert.Equal(t, "0.1.0", summary.Version)
	assert.Equal(t, "uuid-1", summary.ReleaseID)
	assert.NotEmpty(t, summary.Age)
}

func TestRenderReleaseListOutput_InvalidJSONPathNotNeeded(t *testing.T) {
	err := RenderReleaseListOutput([]ReleaseSummary{{Name: "demo"}}, "json", false)
	require.NoError(t, err)
}

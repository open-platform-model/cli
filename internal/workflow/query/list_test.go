package query

import (
	"testing"
	"time"

	"github.com/opmodel/cli/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildInstanceSummary(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	inv := &inventory.InstanceInventoryRecord{InstanceMetadata: inventory.InstanceMetadata{InstanceName: "demo", InstanceNamespace: "apps", InstanceID: "uuid-1", LastTransitionTime: now}, ModuleMetadata: inventory.ModuleMetadata{Name: "module-a", Version: "0.1.0"}}
	summary := BuildInstanceSummary(inv)
	assert.Equal(t, "demo", summary.Name)
	assert.Equal(t, "module-a", summary.Module)
	assert.Equal(t, "cli", summary.Owner)
	assert.Equal(t, "0.1.0", summary.Version)
	assert.Equal(t, "uuid-1", summary.InstanceID)
	assert.NotEmpty(t, summary.Age)
}

func TestRenderInstanceListOutput_InvalidJSONPathNotNeeded(t *testing.T) {
	err := RenderInstanceListOutput([]InstanceSummary{{Name: "demo"}}, "json", false)
	require.NoError(t, err)
}

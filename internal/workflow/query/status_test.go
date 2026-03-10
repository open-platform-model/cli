package query

import (
	"testing"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/inventory"
	"github.com/opmodel/cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestParseStatusOutputFormat(t *testing.T) {
	format, err := ParseStatusOutputFormat("wide")
	require.NoError(t, err)
	assert.Equal(t, output.FormatWide, format)
	_, err = ParseStatusOutputFormat("dir")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestBuildStatusOptions(t *testing.T) {
	rsf := &cmdutil.ReleaseSelectorFlags{ReleaseName: "demo", ReleaseID: "uuid-1"}
	inv := &inventory.InventorySecret{Index: []string{"change-1"}, Changes: map[string]*inventory.ChangeEntry{"change-1": {Source: inventory.ChangeSource{Version: "1.2.3"}, Inventory: inventory.InventoryList{Entries: []inventory.InventoryEntry{{Kind: "Service", Namespace: "apps", Name: "web", Component: "frontend"}}}}}}
	live := []*unstructured.Unstructured{{}}
	missing := []inventory.InventoryEntry{{Kind: "ConfigMap", Namespace: "apps", Name: "cfg"}}
	opts := BuildStatusOptions("apps", rsf, output.FormatWide, true, inv, live, missing)
	assert.Equal(t, "apps", opts.Namespace)
	assert.Equal(t, "1.2.3", opts.Version)
	assert.True(t, opts.Wide)
	assert.True(t, opts.Verbose)
	assert.Equal(t, "frontend", opts.ComponentMap["Service/apps/web"])
	require.Len(t, opts.MissingResources, 1)
	assert.Equal(t, "ConfigMap", opts.MissingResources[0].Kind)
}

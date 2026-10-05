package controller

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/controller/crds"
	"github.com/open-platform-model/cli/internal/inventory"
)

// The embedded manifests are the CRDs the pinned module version renders for
// the controller's instance: the four CRDs install knows, labeled with the
// pinned version and the instance, OPM-managed so a later full install's
// guard accepts them, and the ModuleInstance CRD carries this cli's floor.
// task controller:pin writes them; the pin check (G1) compares them with a
// fresh render.
func TestEmbeddedCRDs_MatchThePin(t *testing.T) {
	objs, err := crds.Objects()
	require.NoError(t, err)

	names := make([]string, 0, len(objs))
	for _, obj := range objs {
		names = append(names, obj.GetName())
		labels := obj.GetLabels()
		assert.Equal(t, PinnedModuleVersion, labels["module.opmodel.dev/version"], "%s: rendered from the pinned module version", obj.GetName())
		assert.Equal(t, "opm_controller", labels["module.opmodel.dev/name"], obj.GetName())
		assert.Equal(t, ControllerInstanceName, labels["module-instance.opmodel.dev/name"], obj.GetName())
		assert.Equal(t, "opm-cli", labels["app.kubernetes.io/managed-by"], obj.GetName())
	}
	want := CRDNames()
	sort.Strings(want)
	assert.Equal(t, want, names)

	mi := renderedCRD(objs, inventory.CRDNameModuleInstances)
	require.NotNil(t, mi)
	assert.NoError(t, inventory.CheckCRDFieldFloor(mi))
}

// Each embedded file is exactly what the pin task writes for the object it
// holds, so a re-pin of the same render changes no byte.
func TestEmbeddedCRDs_AreCanonical(t *testing.T) {
	objs, err := crds.Objects()
	require.NoError(t, err)
	for _, obj := range objs {
		want, err := crds.Manifest(obj)
		require.NoError(t, err)
		got, err := os.ReadFile(filepath.Join("crds", crds.FileName(obj)))
		require.NoError(t, err)
		assert.Equal(t, string(want), string(got), crds.FileName(obj))
	}
}

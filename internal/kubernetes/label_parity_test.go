package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"

	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

// TestLabelValuesMatchRetiredCopy pins the OPM label keys and managed-by
// values the cli writes to and reads from a cluster. The literals are the
// values pkg/core carried before the label vocabulary moved to the library's
// opm/k8s/labels (0012:D1); a library release that changes one fails here.
func TestLabelValuesMatchRetiredCopy(t *testing.T) {
	for _, tc := range []struct {
		name     string
		want     string
		lib, old string
	}{
		{"ManagedBy", "app.kubernetes.io/managed-by", opmlabels.ManagedBy, pkgcore.LabelManagedBy},
		{"ManagedByCLI", "opm-cli", opmlabels.ManagedByCLI, pkgcore.LabelManagedByValue},
		{"ManagedByController", "opm-controller", opmlabels.ManagedByController, pkgcore.LabelManagedByControllerValue},
		{"ManagedByLegacy", "open-platform-model", opmlabels.ManagedByLegacy, pkgcore.LabelManagedByLegacyValue},
		{"Component", "opmodel.dev/component", opmlabels.Component, pkgcore.LabelComponent},
		{"ComponentName", "component.opmodel.dev/name", opmlabels.ComponentName, pkgcore.LabelComponentName},
		{"ModuleInstanceName", "module-instance.opmodel.dev/name", opmlabels.ModuleInstanceName, pkgcore.LabelModuleInstanceName},
		{"ModuleInstanceNamespace", "module-instance.opmodel.dev/namespace", opmlabels.ModuleInstanceNamespace, pkgcore.LabelModuleInstanceNamespace},
		{"ModuleInstanceUUID", "module-instance.opmodel.dev/uuid", opmlabels.ModuleInstanceUUID, pkgcore.LabelModuleInstanceUUID},
	} {
		assert.Equal(t, tc.want, tc.lib, "opmlabels.%s", tc.name)
		assert.Equal(t, tc.want, tc.old, "pkg/core value for %s", tc.name)
	}

	for _, v := range []string{"opm-cli", "opm-controller", "open-platform-model"} {
		assert.True(t, opmlabels.IsOPMManagedBy(v), "opmlabels.IsOPMManagedBy(%q)", v)
		assert.True(t, pkgcore.IsOPMManagedBy(v), "pkgcore.IsOPMManagedBy(%q)", v)
	}
	assert.False(t, opmlabels.IsOPMManagedBy("helm"))
	assert.False(t, pkgcore.IsOPMManagedBy("helm"))
}

package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
)

// TestLabelValuesMatchRetiredCopy pins the OPM label keys and managed-by
// values the cli writes to and reads from a cluster. The literals are the
// values the cli's own pkg/core carried before the label vocabulary moved to
// the library's opm/k8s/labels (0012:D1); while both existed, this test
// asserted each literal against both. A library release that changes one
// fails here.
func TestLabelValuesMatchRetiredCopy(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
		lib  string
	}{
		{"ManagedBy", "app.kubernetes.io/managed-by", opmlabels.ManagedBy},
		{"ManagedByCLI", "opm-cli", opmlabels.ManagedByCLI},
		{"ManagedByController", "opm-controller", opmlabels.ManagedByController},
		{"ManagedByLegacy", "open-platform-model", opmlabels.ManagedByLegacy},
		{"Component", "opmodel.dev/component", opmlabels.Component},
		{"ComponentName", "component.opmodel.dev/name", opmlabels.ComponentName},
		{"ModuleInstanceName", "module-instance.opmodel.dev/name", opmlabels.ModuleInstanceName},
		{"ModuleInstanceNamespace", "module-instance.opmodel.dev/namespace", opmlabels.ModuleInstanceNamespace},
		{"ModuleInstanceUUID", "module-instance.opmodel.dev/uuid", opmlabels.ModuleInstanceUUID},
	} {
		assert.Equal(t, tc.want, tc.lib, "opmlabels.%s", tc.name)
	}

	for _, v := range []string{"opm-cli", "opm-controller", "open-platform-model"} {
		assert.True(t, opmlabels.IsOPMManagedBy(v), "opmlabels.IsOPMManagedBy(%q)", v)
	}
	assert.False(t, opmlabels.IsOPMManagedBy("helm"))
}

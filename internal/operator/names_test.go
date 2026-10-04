package operator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/cli/internal/inventory"
)

func crdEntry(name string) inventory.InventoryEntry {
	return inventory.InventoryEntry{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: name}
}

func TestDeploysOperator(t *testing.T) {
	tests := []struct {
		name       string
		rec        *inventory.Record
		wantSignal string
		wantOK     bool
	}{
		{
			name:       "coordinates with any module path",
			rec:        &inventory.Record{Name: "opm-operator", Namespace: "opm-operator-system", ModulePath: "example.com/modules/other@v3"},
			wantSignal: SignalCoordinates, wantOK: true,
		},
		{
			name:       "coordinates with no module path",
			rec:        &inventory.Record{Name: "opm-operator", Namespace: "opm-operator-system"},
			wantSignal: SignalCoordinates, wantOK: true,
		},
		{
			name:       "module path v0",
			rec:        &inventory.Record{Name: "team-ops", Namespace: "platform", ModulePath: "opmodel.dev/modules/opm_operator@v0"},
			wantSignal: SignalModulePath, wantOK: true,
		},
		{
			name:       "module path v1",
			rec:        &inventory.Record{Name: "team-ops", Namespace: "platform", ModulePath: "opmodel.dev/modules/opm_operator@v1"},
			wantSignal: SignalModulePath, wantOK: true,
		},
		{
			name:       "module path without a major",
			rec:        &inventory.Record{Name: "team-ops", Namespace: "platform", ModulePath: "opmodel.dev/modules/opm_operator"},
			wantSignal: SignalModulePath, wantOK: true,
		},
		{
			name: "inventory holds an operator CRD",
			rec: &inventory.Record{
				Name: "crds", Namespace: "platform", ModulePath: "example.com/modules/crds@v0",
				Inventory: inventory.Inventory{Entries: []inventory.InventoryEntry{
					{Kind: "ConfigMap", Name: "x", Namespace: "platform"},
					crdEntry("moduleinstances.opmodel.dev"),
				}},
			},
			wantSignal: SignalInventoryCRD, wantOK: true,
		},
		{
			name: "look-alike name, path and CRD",
			rec: &inventory.Record{
				Name: "opm-operator", Namespace: "default", ModulePath: "opmodel.dev/modules/opm_operator_dashboard@v0",
				Inventory: inventory.Inventory{Entries: []inventory.InventoryEntry{crdEntry("widgets.example.opmodel.dev.io")}},
			},
		},
		{
			name: "another instance in the operator's namespace",
			rec:  &inventory.Record{Name: "hello", Namespace: "opm-operator-system", ModulePath: "example.com/modules/hello@v0"},
		},
		{
			name: "a CRD of a subgroup of opmodel.dev",
			rec: &inventory.Record{
				Name: "sub", Namespace: "default",
				Inventory: inventory.Inventory{Entries: []inventory.InventoryEntry{crdEntry("widgets.example.opmodel.dev")}},
			},
		},
		{
			name: "an opmodel.dev object that is not a CRD",
			rec: &inventory.Record{
				Name: "plat", Namespace: "default",
				Inventory: inventory.Inventory{Entries: []inventory.InventoryEntry{
					{Group: "opmodel.dev", Kind: "Platform", Name: "cluster.opmodel.dev"},
				}},
			},
		},
		{name: "no record", rec: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal, ok := DeploysOperator(tt.rec)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantSignal, signal)
		})
	}
}

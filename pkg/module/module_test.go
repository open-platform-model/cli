package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalModuleRef(t *testing.T) {
	cases := []struct {
		name        string
		meta        ModuleMetadata
		wantPath    string
		wantVersion string
	}{
		{
			name:        "nameSnakeCase present",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules", Name: "cert-manager", NameSnakeCase: "cert_manager", Version: "0.1.0"},
			wantPath:    "opmodel.dev/modules/cert_manager@v0",
			wantVersion: "0.1.0",
		},
		{
			name:        "nameSnakeCase derived from kebab name",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules", Name: "zot-registry-ttl", Version: "1.2.3"},
			wantPath:    "opmodel.dev/modules/zot_registry_ttl@v1",
			wantVersion: "1.2.3",
		},
		{
			name:        "already-snake name",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules", Name: "metallb", Version: "0.2.1"},
			wantPath:    "opmodel.dev/modules/metallb@v0",
			wantVersion: "0.2.1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, version := tc.meta.CanonicalModuleRef()
			assert.Equal(t, tc.wantPath, path)
			assert.Equal(t, tc.wantVersion, version)
		})
	}
}

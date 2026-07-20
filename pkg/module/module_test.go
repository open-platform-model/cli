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
			wantVersion: "v0.1.0",
		},
		{
			name:        "nameSnakeCase derived from kebab name",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules", Name: "zot-registry-ttl", Version: "1.2.3"},
			wantPath:    "opmodel.dev/modules/zot_registry_ttl@v1",
			wantVersion: "v1.2.3",
		},
		{
			name:        "already-snake name",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules", Name: "metallb", Version: "0.2.1"},
			wantPath:    "opmodel.dev/modules/metallb@v0",
			wantVersion: "v0.2.1",
		},
		{
			// A module that already declares the prefix must not gain a second
			// one — the normalization is idempotent.
			name:        "version already v-prefixed",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules", Name: "podinfo", Version: "v0.1.3"},
			wantPath:    "opmodel.dev/modules/podinfo@v0",
			wantVersion: "v0.1.3",
		},
		{
			// An uppercase prefix must be recognized by both helpers or one
			// strips it and the other does not, yielding a "vV0" major tag.
			name:        "version prefixed with an uppercase V",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules", Name: "podinfo", Version: "V0.1.3"},
			wantPath:    "opmodel.dev/modules/podinfo@v0",
			wantVersion: "V0.1.3",
		},
		{
			// No declared version stays empty so callers can still detect it,
			// rather than becoming a bare "v".
			name:        "empty version stays empty",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules", Name: "podinfo"},
			wantPath:    "opmodel.dev/modules/podinfo@v0",
			wantVersion: "",
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

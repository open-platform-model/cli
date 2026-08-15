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
			// Core v2: metadata.modulePath is the complete registry address;
			// the CLI composes nothing on top of it.
			name:        "path read verbatim",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules/cert_manager@v0", Name: "cert_manager", Version: "0.1.0"},
			wantPath:    "opmodel.dev/modules/cert_manager@v0",
			wantVersion: "v0.1.0",
		},
		{
			name:        "bare semver gains the v prefix",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules/zot_registry_ttl@v1", Name: "zot_registry_ttl", Version: "1.2.3"},
			wantPath:    "opmodel.dev/modules/zot_registry_ttl@v1",
			wantVersion: "v1.2.3",
		},
		{
			// A module that already declares the prefix must not gain a second
			// one — the normalization is idempotent.
			name:        "version already v-prefixed",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules/podinfo@v0", Name: "podinfo", Version: "v0.1.4"},
			wantPath:    "opmodel.dev/modules/podinfo@v0",
			wantVersion: "v0.1.4",
		},
		{
			// An uppercase prefix is recognized as already-prefixed rather than
			// gaining a second "v".
			name:        "version prefixed with an uppercase V",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules/podinfo@v0", Name: "podinfo", Version: "V0.1.4"},
			wantPath:    "opmodel.dev/modules/podinfo@v0",
			wantVersion: "V0.1.4",
		},
		{
			// No declared version stays empty so callers can still detect it,
			// rather than becoming a bare "v".
			name:        "empty version stays empty",
			meta:        ModuleMetadata{ModulePath: "opmodel.dev/modules/podinfo@v0", Name: "podinfo"},
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

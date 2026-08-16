package cueedit

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeIdentity creates dir/identity/identity.cue with the given content and
// returns dir.
func writeIdentity(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "identity"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "identity", "identity.cue"), []byte(content), 0o644))
	return dir
}

func readIdentity(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "identity", "identity.cue"))
	require.NoError(t, err)
	return string(data)
}

// evalVersion evaluates the rewritten file (with defaults) and returns the
// concrete Version string.
func evalVersion(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "identity", "identity.cue"))
	require.NoError(t, err)
	v := cuecontext.New().CompileBytes(data)
	require.NoError(t, v.Err())
	s, err := v.LookupPath(cue.ParsePath("Version")).String()
	require.NoError(t, err)
	return s
}

func TestSetIdentityVersion(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		version string
		want    string // full expected file content after the rewrite
	}{
		{
			name: "literal value replaced",
			in: `package identity

ModulePath: "opmodel.dev/modules/web_app@v1"
Version:    "1.0.0"
`,
			version: "1.2.0",
			want: `package identity

ModulePath: "opmodel.dev/modules/web_app@v1"
Version:    "1.2.0"
`,
		},
		{
			name: "defaulted value rewrites the default",
			in: `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

ModulePath: "opmodel.dev/catalogs/opm@v2"
Version:    #VersionType | *"2.0.0-alpha.3" // x-release-please-version
`,
			version: "2.0.0-alpha.4",
			want: `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

ModulePath: "opmodel.dev/catalogs/opm@v2"
Version:    #VersionType | *"2.0.0-alpha.4" // x-release-please-version
`,
		},
		{
			name: "open field gains a unification",
			in: `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

ModulePath: "opmodel.dev/modules/web_app@v1"
Version:    #VersionType
`,
			version: "1.3.0",
			want: `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

ModulePath: "opmodel.dev/modules/web_app@v1"
Version:    #VersionType & "1.3.0"
`,
		},
		{
			name: "open disjunction parenthesized before unifying",
			in: `package identity

Version: string | =~"^\\d"
`,
			version: "1.0.0",
			want: `package identity

Version: (string | =~"^\\d") & "1.0.0"
`,
		},
		{
			name: "literal inside an and-chain replaced",
			in: `package identity

#VersionType: string

Version: #VersionType & "0.9.0"
`,
			version: "0.9.1",
			want: `package identity

#VersionType: string

Version: #VersionType & "0.9.1"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeIdentity(t, tt.in)
			require.NoError(t, SetIdentityVersion(dir, tt.version))
			assert.Equal(t, tt.want, readIdentity(t, dir))
			assert.Equal(t, tt.version, evalVersion(t, dir))
		})
	}
}

func TestSetIdentityVersion_ShapeRefusals(t *testing.T) {
	t.Run("absent file", func(t *testing.T) {
		dir := t.TempDir()
		err := SetIdentityVersion(dir, "1.0.0")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIdentityShape)
		assert.Contains(t, err.Error(), "identity.cue")
	})

	t.Run("missing Version field", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

ModulePath: "opmodel.dev/modules/web_app@v1"
`)
		err := SetIdentityVersion(dir, "1.0.0")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIdentityShape)
		assert.Contains(t, err.Error(), "Version")
	})

	t.Run("renamed field is not searched for", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

ModulePath:     "opmodel.dev/catalogs/opm@v2"
CatalogVersion: "2.0.0"
`)
		err := SetIdentityVersion(dir, "1.0.0")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIdentityShape)
	})

	t.Run("unparseable file", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

Version: "1.0.0
`)
		err := SetIdentityVersion(dir, "1.0.0")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIdentityShape)
	})

	t.Run("invalid version rejected before any read", func(t *testing.T) {
		err := SetIdentityVersion(t.TempDir(), "v1.0.0")
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrIdentityShape)
		assert.Contains(t, err.Error(), "SemVer")
	})
}

// TestSetIdentityVersion_PreservesCommentsAndAlignment is the golden test for
// the surgical property: every byte outside the version literal survives,
// including doc comments, inline comments, alignment, and field order.
func TestSetIdentityVersion_PreservesCommentsAndAlignment(t *testing.T) {
	in := `// Package identity is the single source of this module's path and version
// (core #IdentityPackage, enhancements 0010 D38 / 0011 D12).
package identity

// #VersionType mirrors core.#VersionType (SemVer 2.0), duplicated so this
// package stays import-free.
#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"

// ModulePath is the module's complete CUE module path, major suffix included.
ModulePath: "opmodel.dev/modules/web_app@v1"

// Version is the module's bare SemVer; its major must agree with ModulePath's.
Version: #VersionType | *"1.0.0"
`
	want := `// Package identity is the single source of this module's path and version
// (core #IdentityPackage, enhancements 0010 D38 / 0011 D12).
package identity

// #VersionType mirrors core.#VersionType (SemVer 2.0), duplicated so this
// package stays import-free.
#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"

// ModulePath is the module's complete CUE module path, major suffix included.
ModulePath: "opmodel.dev/modules/web_app@v1"

// Version is the module's bare SemVer; its major must agree with ModulePath's.
Version: #VersionType | *"1.1.0"
`

	dir := writeIdentity(t, in)
	require.NoError(t, SetIdentityVersion(dir, "1.1.0"))
	assert.Equal(t, want, readIdentity(t, dir))
}

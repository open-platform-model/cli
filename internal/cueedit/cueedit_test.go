package cueedit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
			changed, err := SetIdentityVersion(dir, tt.version)
			require.NoError(t, err)
			assert.True(t, changed)
			assert.Equal(t, tt.want, readIdentity(t, dir))
			assert.Equal(t, tt.version, evalVersion(t, dir))
		})
	}
}

// TestSetIdentityVersion_NoOp is D3's idempotency contract: setting the
// version the file already declares writes nothing — identical bytes, no
// mtime change — and reports changed false.
func TestSetIdentityVersion_NoOp(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		version string
	}{
		{
			name: "literal already set",
			in: `package identity

ModulePath: "opmodel.dev/modules/web_app@v1"
Version:    "1.2.0"
`,
			version: "1.2.0",
		},
		{
			name: "default already set",
			in: `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

ModulePath: "opmodel.dev/catalogs/opm@v2"
Version:    #VersionType | *"2.0.0-alpha.3" // x-release-please-version
`,
			version: "2.0.0-alpha.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeIdentity(t, tt.in)
			path := filepath.Join(dir, "identity", "identity.cue")

			// Backdate the mtime so an accidental rewrite is detectable even
			// on filesystems with coarse timestamp resolution.
			past := time.Now().Add(-time.Hour)
			require.NoError(t, os.Chtimes(path, past, past))
			before, err := os.Stat(path)
			require.NoError(t, err)

			changed, err := SetIdentityVersion(dir, tt.version)
			require.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, tt.in, readIdentity(t, dir))

			after, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, before.ModTime(), after.ModTime())
		})
	}
}

func TestReadIdentityVersion(t *testing.T) {
	tests := []struct {
		name          string
		in            string
		wantValue     string
		wantDefaulted bool
	}{
		{
			name: "literal value",
			in: `package identity

Version: "1.0.0"
`,
			wantValue:     "1.0.0",
			wantDefaulted: false,
		},
		{
			name: "literal in an and-chain",
			in: `package identity

#VersionType: string

Version: #VersionType & "0.9.0"
`,
			wantValue:     "0.9.0",
			wantDefaulted: false,
		},
		{
			name: "defaulted disjunction",
			in: `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

Version: #VersionType | *"2.0.0-alpha.3" // x-release-please-version
`,
			wantValue:     "2.0.0-alpha.3",
			wantDefaulted: true,
		},
		{
			name: "open field reads as empty",
			in: `package identity

#VersionType: string

Version: #VersionType
`,
			wantValue:     "",
			wantDefaulted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeIdentity(t, tt.in)
			value, defaulted, err := ReadIdentityVersion(dir)
			require.NoError(t, err)
			assert.Equal(t, tt.wantValue, value)
			assert.Equal(t, tt.wantDefaulted, defaulted)
		})
	}
}

func TestReadIdentityVersion_ShapeRefusals(t *testing.T) {
	t.Run("absent file", func(t *testing.T) {
		_, _, err := ReadIdentityVersion(t.TempDir())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIdentityShape)
	})

	t.Run("missing Version field", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

ModulePath: "opmodel.dev/modules/web_app@v1"
`)
		_, _, err := ReadIdentityVersion(dir)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIdentityShape)
		assert.Contains(t, err.Error(), "Version")
	})

	t.Run("unparseable file", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

Version: "1.0.0
`)
		_, _, err := ReadIdentityVersion(dir)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIdentityShape)
	})
}

func TestSetIdentityVersion_ShapeRefusals(t *testing.T) {
	t.Run("absent file", func(t *testing.T) {
		dir := t.TempDir()
		_, err := SetIdentityVersion(dir, "1.0.0")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIdentityShape)
		assert.Contains(t, err.Error(), "identity.cue")
	})

	t.Run("missing Version field", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

ModulePath: "opmodel.dev/modules/web_app@v1"
`)
		_, err := SetIdentityVersion(dir, "1.0.0")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIdentityShape)
		assert.Contains(t, err.Error(), "Version")
	})

	t.Run("renamed field is not searched for", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

ModulePath:     "opmodel.dev/catalogs/opm@v2"
CatalogVersion: "2.0.0"
`)
		_, err := SetIdentityVersion(dir, "1.0.0")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIdentityShape)
	})

	t.Run("unparseable file", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

Version: "1.0.0
`)
		_, err := SetIdentityVersion(dir, "1.0.0")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIdentityShape)
	})

	t.Run("invalid version rejected before any read", func(t *testing.T) {
		_, err := SetIdentityVersion(t.TempDir(), "v1.0.0")
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
	changed, err := SetIdentityVersion(dir, "1.1.0")
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, want, readIdentity(t, dir))
}

// writeCueMod creates dir/cue.mod/module.cue with the given content and
// returns dir.
func writeCueMod(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(content), 0o644))
	return dir
}

func readCueMod(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	return string(data)
}

func TestSetCueModModule(t *testing.T) {
	t.Run("module line replaced, everything else preserved", func(t *testing.T) {
		in := `// This file declares the module's registry address (0010 D1: byte-identical
// to identity's ModulePath).
module: "opmodel.dev/modules/web_app@v1" // trailing comment survives

language: {
	version: "v0.17.0"
}
`
		want := `// This file declares the module's registry address (0010 D1: byte-identical
// to identity's ModulePath).
module: "opmodel.dev/modules/web_app@v2" // trailing comment survives

language: {
	version: "v0.17.0"
}
`
		dir := writeCueMod(t, in)
		changed, err := SetCueModModule(dir, "opmodel.dev/modules/web_app@v2")
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, want, readCueMod(t, dir))
	})

	t.Run("no-op leaves bytes and mtime untouched", func(t *testing.T) {
		in := `module: "opmodel.dev/modules/web_app@v1"

language: version: "v0.17.0"
`
		dir := writeCueMod(t, in)
		path := filepath.Join(dir, "cue.mod", "module.cue")
		past := time.Now().Add(-time.Hour)
		require.NoError(t, os.Chtimes(path, past, past))
		before, err := os.Stat(path)
		require.NoError(t, err)

		changed, err := SetCueModModule(dir, "opmodel.dev/modules/web_app@v1")
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, in, readCueMod(t, dir))

		after, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, before.ModTime(), after.ModTime())
	})

	t.Run("absent file", func(t *testing.T) {
		_, err := SetCueModModule(t.TempDir(), "opmodel.dev/modules/web_app@v1")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCueModShape)
		assert.Contains(t, err.Error(), "module.cue")
	})

	t.Run("missing module field", func(t *testing.T) {
		dir := writeCueMod(t, `language: version: "v0.17.0"
`)
		_, err := SetCueModModule(dir, "opmodel.dev/modules/web_app@v1")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCueModShape)
		assert.Contains(t, err.Error(), "module")
	})

	t.Run("non-literal module value", func(t *testing.T) {
		dir := writeCueMod(t, `module: _path

_path: "opmodel.dev/modules/web_app@v1"
`)
		_, err := SetCueModModule(dir, "opmodel.dev/modules/web_app@v1")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCueModShape)
	})

	t.Run("unparseable file", func(t *testing.T) {
		dir := writeCueMod(t, `module: "opmodel.dev
`)
		_, err := SetCueModModule(dir, "opmodel.dev/modules/web_app@v1")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCueModShape)
	})

	t.Run("empty path rejected before any read", func(t *testing.T) {
		_, err := SetCueModModule(t.TempDir(), "")
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrCueModShape)
		assert.Contains(t, err.Error(), "empty")
	})
}

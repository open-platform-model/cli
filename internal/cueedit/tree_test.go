package cueedit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTree creates a module tree from relative-path → content and returns
// its root.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return dir
}

func readFile(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, rel))
	require.NoError(t, err)
	return string(data)
}

func TestRewriteSelfImports(t *testing.T) {
	const oldPath = "opmodel.dev/templates/standard@v1"
	const newPath = "example.com/modules/my_app@v0"

	t.Run("identity import line across multiple files", func(t *testing.T) {
		dir := writeTree(t, map[string]string{
			"cue.mod/module.cue": `module: "opmodel.dev/templates/standard@v1"` + "\n",
			"module.cue": `package standard

import (
	m "opmodel.dev/core@v2"

	id "opmodel.dev/templates/standard/identity"
)

m.#Module
metadata: modulePath: id.ModulePath
`,
			"components.cue": `package standard

import (
	tr "opmodel.dev/catalogs/opm/traits/v1beta1"

	id "opmodel.dev/templates/standard/identity" // version label source
)

_v: id.Version
_t: tr.#Expose
`,
			"identity/identity.cue": `package identity

ModulePath: "opmodel.dev/templates/standard@v1"
`,
		})

		changed, err := RewriteSelfImports(dir, oldPath, newPath)
		require.NoError(t, err)
		assert.Equal(t, []string{"components.cue", "module.cue"}, changed)

		assert.Equal(t, `package standard

import (
	m "opmodel.dev/core@v2"

	id "example.com/modules/my_app/identity"
)

m.#Module
metadata: modulePath: id.ModulePath
`, readFile(t, dir, "module.cue"))

		// The alias and the trailing comment survive; the foreign import does.
		components := readFile(t, dir, "components.cue")
		assert.Contains(t, components, `id "example.com/modules/my_app/identity" // version label source`)
		assert.Contains(t, components, `tr "opmodel.dev/catalogs/opm/traits/v1beta1"`)

		// cue.mod and non-importing files are untouched.
		assert.Contains(t, readFile(t, dir, "cue.mod/module.cue"), "templates/standard")
		assert.Contains(t, readFile(t, dir, "identity/identity.cue"), "templates/standard")
	})

	t.Run("module-root import and a majored self-import follow the new major", func(t *testing.T) {
		dir := writeTree(t, map[string]string{
			"extra.cue": `package standard

import (
	self "opmodel.dev/templates/standard@v1:standard"
	sub "opmodel.dev/templates/standard/identity@v1"
)

_a: self
_b: sub
`,
		})

		changed, err := RewriteSelfImports(dir, oldPath, newPath)
		require.NoError(t, err)
		assert.Equal(t, []string{"extra.cue"}, changed)

		out := readFile(t, dir, "extra.cue")
		assert.Contains(t, out, `self "example.com/modules/my_app@v0:standard"`)
		assert.Contains(t, out, `sub "example.com/modules/my_app/identity@v0"`)
	})

	t.Run("prefix match is per segment, not per byte", func(t *testing.T) {
		dir := writeTree(t, map[string]string{
			"module.cue": `package standard

import other "opmodel.dev/templates/standard_extras/lib"

_o: other
`,
		})

		changed, err := RewriteSelfImports(dir, oldPath, newPath)
		require.NoError(t, err)
		assert.Empty(t, changed)
	})

	t.Run("no self-imports is a clean no-op", func(t *testing.T) {
		dir := writeTree(t, map[string]string{
			"module.cue": "package standard\n\nx: 1\n",
		})
		changed, err := RewriteSelfImports(dir, oldPath, newPath)
		require.NoError(t, err)
		assert.Empty(t, changed)
	})
}

func TestRenamePackageClauses(t *testing.T) {
	t.Run("all files renamed, comments and whitespace preserved", func(t *testing.T) {
		dir := writeTree(t, map[string]string{
			"cue.mod/module.cue": `module: "opmodel.dev/templates/standard@v1"` + "\n",
			"module.cue": `// Package standard is the template.
package standard // the clause comment survives

x: 1
`,
			"components.cue": `package standard

y: 2
`,
			"identity/identity.cue": `package identity

ModulePath: "opmodel.dev/templates/standard@v1"
`,
		})

		changed, err := RenamePackageClauses(dir, "standard", "my_app")
		require.NoError(t, err)
		assert.Equal(t, []string{"components.cue", "module.cue"}, changed)

		assert.Equal(t, `// Package standard is the template.
package my_app // the clause comment survives

x: 1
`, readFile(t, dir, "module.cue"))
		assert.Equal(t, "package my_app\n\ny: 2\n", readFile(t, dir, "components.cue"))

		// The identity subpackage keeps its own name.
		assert.Contains(t, readFile(t, dir, "identity/identity.cue"), "package identity")
	})

	t.Run("other package names untouched", func(t *testing.T) {
		dir := writeTree(t, map[string]string{
			"module.cue": "package other\n\nx: 1\n",
		})
		changed, err := RenamePackageClauses(dir, "standard", "my_app")
		require.NoError(t, err)
		assert.Empty(t, changed)
	})
}

func TestSetIdentityModulePath(t *testing.T) {
	t.Run("literal replaced, surrounding bytes preserved", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

// ModulePath is byte-identical to cue.mod's module field.
ModulePath: "opmodel.dev/templates/standard@v1" // trailing comment

Version: "1.0.0"
`)
		changed, err := SetIdentityModulePath(dir, "example.com/modules/my_app@v0")
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, `package identity

// ModulePath is byte-identical to cue.mod's module field.
ModulePath: "example.com/modules/my_app@v0" // trailing comment

Version: "1.0.0"
`, readIdentity(t, dir))
	})

	t.Run("idempotent no-op", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

ModulePath: "example.com/modules/my_app@v0"
Version:    "1.0.0"
`)
		changed, err := SetIdentityModulePath(dir, "example.com/modules/my_app@v0")
		require.NoError(t, err)
		assert.False(t, changed)
	})

	t.Run("non-literal value is a shape refusal", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

ModulePath: _base + "@v1"
Version:    "1.0.0"
`)
		_, err := SetIdentityModulePath(dir, "example.com/modules/my_app@v0")
		require.ErrorIs(t, err, ErrIdentityShape)
	})

	t.Run("missing field is a shape refusal", func(t *testing.T) {
		dir := writeIdentity(t, "package identity\n\nVersion: \"1.0.0\"\n")
		_, err := SetIdentityModulePath(dir, "example.com/modules/my_app@v0")
		require.ErrorIs(t, err, ErrIdentityShape)
	})
}

func TestResetIdentityVersion(t *testing.T) {
	t.Run("defaulted disjunction keeps its shape and marker", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

ModulePath: "opmodel.dev/templates/standard@v1"
Version:    #VersionType | *"1.4.0" // x-release-please-version
`)
		changed, err := ResetIdentityVersion(dir, "0.1.0")
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

ModulePath: "opmodel.dev/templates/standard@v1"
Version:    #VersionType | *"0.1.0" // x-release-please-version
`, readIdentity(t, dir))
	})

	t.Run("concrete literal becomes the defaulted form", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

ModulePath: "opmodel.dev/modules/donor@v2"
Version:    "2.3.1"
`)
		changed, err := ResetIdentityVersion(dir, "0.1.0")
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Contains(t, readIdentity(t, dir), `Version:    #VersionType | *"0.1.0"`)
		assert.Equal(t, "0.1.0", evalVersion(t, dir))
	})

	t.Run("no #VersionType falls back to an in-place literal replace", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

ModulePath: "opmodel.dev/modules/donor@v2"
Version:    "2.3.1"
`)
		changed, err := ResetIdentityVersion(dir, "0.1.0")
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Contains(t, readIdentity(t, dir), `Version:    "0.1.0"`)
	})

	t.Run("already reset is a no-op", func(t *testing.T) {
		dir := writeIdentity(t, `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

ModulePath: "example.com/modules/my_app@v0"
Version:    #VersionType | *"0.1.0" // x-release-please-version
`)
		changed, err := ResetIdentityVersion(dir, "0.1.0")
		require.NoError(t, err)
		assert.False(t, changed)
	})
}

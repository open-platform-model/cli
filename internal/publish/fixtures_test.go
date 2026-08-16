package publish

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelabs.dev/go/oci/ociregistry/ocimem"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/require"
)

// identitySchemaStub mirrors core.#IdentityPackage's authored surface for
// hermetic unit tests: two required fields, closed, with the catalog's
// derived declarations admitted. The real definition is exercised by the
// registry-backed conformance test, which skips when the schema cannot be
// resolved.
const identitySchemaStub = `
#IdentityPackage: {
	ModulePath!:   string & =~"^[a-z0-9._-]+(/[a-z0-9._-]+)*@v[0-9]+$"
	Version!:      string & =~"^\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
	RegistryPath?: string
	Major?:        string
	kindPrefix?: [string]: string
}
`

// stubSchema compiles the test #IdentityPackage definition.
func stubSchema(t *testing.T, ctx *cue.Context) cue.Value {
	t.Helper()
	v := ctx.CompileString(identitySchemaStub)
	require.NoError(t, v.Err())
	def := v.LookupPath(cue.MakePath(cue.Def("IdentityPackage")))
	require.True(t, def.Exists())
	return def
}

// moduleFiles is a minimal conformant module tree with no external imports —
// loads hermetically, passes every gate.
func moduleFiles() map[string]string {
	return map[string]string{
		"cue.mod/module.cue": `module: "example.com/modules/demo@v1"
language: version: "v0.17.0"
source: kind: "self"
`,
		"identity/identity.cue": `package identity

ModulePath: "example.com/modules/demo@v1"
Version:    "1.2.0"
`,
		"module.cue": `package demo

metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v1"
	version:    "1.2.0"
}
`,
	}
}

// catalogFiles is the catalog twin: identity carries the derived
// declarations, the root package is not named after metadata.name.
func catalogFiles() map[string]string {
	return map[string]string{
		"cue.mod/module.cue": `module: "example.com/catalogs/demo@v1"
language: version: "v0.17.0"
source: kind: "self"
`,
		"identity/identity.cue": `package identity

ModulePath:   "example.com/catalogs/demo@v1"
Version:      "1.2.0"
RegistryPath: "example.com/catalogs/demo"
`,
		"catalog.cue": `package democat

metadata: {
	name:       "demo"
	modulePath: "example.com/catalogs/demo@v1"
	version:    "1.2.0"
}
`,
	}
}

// writeTree materializes files under a temp dir and returns it.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return dir
}

// edit returns files with one entry replaced (or added).
func edit(files map[string]string, name, content string) map[string]string {
	out := make(map[string]string, len(files))
	for k, v := range files {
		out[k] = v
	}
	out[name] = content
	return out
}

// emptyTestRegistry starts an in-process, immutable-tag OCI registry with
// nothing in it and returns a CUE registry mapping routing every fixture
// domain at it — the already-published lookup runs for real, off the network.
func emptyTestRegistry(t *testing.T) string {
	t.Helper()
	reg, err := modregistrytest.NewServer(ocimem.NewWithConfig(&ocimem.Config{ImmutableTags: true}), nil)
	require.NoError(t, err)
	t.Cleanup(reg.Close)
	return reg.Host() + "+insecure"
}

// baseOptions builds the Options runDir would run with, for tests that call
// pipeline internals directly rather than through Run.
func baseOptions(t *testing.T, dir string) Options {
	t.Helper()
	ctx := cuecontext.New()
	return Options{
		Dir:            dir,
		Kind:           KindCatalog,
		Context:        ctx,
		IdentitySchema: stubSchema(t, ctx),
		Registry:       emptyTestRegistry(t),
	}
}

// runFixture runs the pipeline on files with the stub schema and an empty
// in-process registry; tests that exercise the registry gates configure
// their own.
func runFixture(t *testing.T, kind Kind, files map[string]string, mutate ...func(*Options)) *Plan {
	t.Helper()
	dir := writeTree(t, files)
	return runDir(t, kind, dir, mutate...)
}

func runDir(t *testing.T, kind Kind, dir string, mutate ...func(*Options)) *Plan {
	t.Helper()
	ctx := cuecontext.New()
	opts := Options{
		Dir:            dir,
		Kind:           kind,
		Context:        ctx,
		IdentitySchema: stubSchema(t, ctx),
		Registry:       emptyTestRegistry(t),
	}
	for _, m := range mutate {
		m(&opts)
	}
	p, err := Run(context.Background(), opts)
	require.NoError(t, err)
	return p
}

// refusalHeadlines joins every refusal headline for containment asserts.
func refusalHeadlines(p *Plan) string {
	var b strings.Builder
	for _, r := range p.Refusals {
		b.WriteString(r.Headline)
		b.WriteString("\n")
	}
	return b.String()
}

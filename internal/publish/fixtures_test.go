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
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/stretchr/testify/require"
)

// identitySchemaStub mirrors the core definitions the pipeline unifies
// against, for hermetic unit tests: #IdentityPackage with core's
// Major and VersionMajor derivations, and the two catalog gates copied from
// core verbatim, including the conditional declaredAPIVersion and the hidden
// rule fields whose measured blind spots the gate tests pin. The real
// definitions are exercised by the registry-backed conformance tests, which
// skip when the registry is unreachable.
const identitySchemaStub = `
import "strings"

#NameType:        string & =~"^[a-z0-9]([a-z0-9-]*[a-z0-9])?$"
#VersionType:     string & =~"^\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
#APIVersionType:  string & =~"^v[0-9]+((alpha|beta)[0-9]+)?$"
#PackagePathType: string & =~"^[a-z0-9._-]+(/[a-z0-9._-]+)*$"
#ContractFQNType: string & =~"^[a-z0-9._-]+(/[a-z0-9._-]+)*/[a-z0-9]([a-z0-9-]*[a-z0-9])?@v[0-9]+((alpha|beta)[0-9]+)?$"
#ImplFQNType:     string & =~"^[a-z0-9._-]+(/[a-z0-9._-]+)*/[a-z0-9]([a-z0-9-]*[a-z0-9])?@\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
#FQNType:         #ContractFQNType | #ImplFQNType

// Major and VersionMajor derive as core derives them, so the version major
// must agree with the path major. An open Version leaves VersionMajor
// incomplete, not wrong: the identity check passes it, because --version
// fills it at publish, and checks the identity again as filled. RegistryPath
// and kindPrefix stay loose; catalog fixtures author kindPrefix explicitly,
// because the member gate reads it off the identity value.
#IdentityPackage: {
	ModulePath!:   string & =~"^[a-z0-9._-]+(/[a-z0-9._-]+)*@v[0-9]+$"
	Version!:      #VersionType
	RegistryPath?: string
	Major:         strings.SplitN(ModulePath, "@", 2)[1]
	VersionMajor:  "v" + strings.SplitN(Version, ".", 2)[0]
	VersionMajor:  Major
	kindPrefix?: [string]: string
}

#CatalogMemberFQNGate: {
	identity!: #IdentityPackage
	kind!:     "resources" | "traits" | "blueprints" | "transformers"
	name!:     #NameType

	declaredFQN!:            #FQNType
	declaredModulePath!:     #PackagePathType
	declaredCatalogVersion!: #VersionType

	declaredAPIVersion?: #APIVersionType
	if kind != "transformers" {
		declaredAPIVersion!: #APIVersionType
	}

	declaredModulePath: [
		if kind == "transformers" {identity.kindPrefix[kind]},
		identity.kindPrefix[kind] + "/" + declaredAPIVersion,
	][0]
	declaredCatalogVersion: identity.Version

	_keyVersion: [
		if kind == "transformers" {identity.Version},
		declaredAPIVersion,
	][0]

	declaredFQN: identity.kindPrefix[kind] + "/" + name + "@" + _keyVersion
}

#TraitOptionalGate: {
	optional: bool

	_stated: "\(optional)"

	_overridable: ((optional & true) != _|_) && ((optional & false) != _|_)
	_overridable: true
}
`

// stubSchema compiles the test #IdentityPackage definition.
func stubSchema(t *testing.T, ctx *cue.Context) cue.Value {
	t.Helper()
	return stubDef(t, ctx, "IdentityPackage")
}

// stubDef compiles the stub and returns one named definition from it.
func stubDef(t *testing.T, ctx *cue.Context, name string) cue.Value {
	t.Helper()
	v := ctx.CompileString(identitySchemaStub)
	require.NoError(t, v.Err())
	def := v.LookupPath(cue.MakePath(cue.Def(name)))
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

kind: "Module"
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

// Authored here because the stub #IdentityPackage does not derive it; the
// real schema does (the member gate reads identity.kindPrefix either way).
kindPrefix: {
	resources:    "example.com/catalogs/demo/resources"
	traits:       "example.com/catalogs/demo/traits"
	blueprints:   "example.com/catalogs/demo/blueprints"
	transformers: "example.com/catalogs/demo/transformers"
}
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
	registry := emptyTestRegistry(t)
	return Options{
		Dir:                     dir,
		Kind:                    KindCatalog,
		Context:                 ctx,
		Kernel:                  kernel.New(kernel.WithRegistry(registry)),
		IdentitySchema:          stubSchema(t, ctx),
		MemberFQNGateSchema:     stubDef(t, ctx, "CatalogMemberFQNGate"),
		TraitOptionalGateSchema: stubDef(t, ctx, "TraitOptionalGate"),
		Registry:                registry,
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
	registry := emptyTestRegistry(t)
	opts := Options{
		Dir:                     dir,
		Kind:                    kind,
		Context:                 ctx,
		Kernel:                  kernel.New(kernel.WithRegistry(registry)),
		IdentitySchema:          stubSchema(t, ctx),
		MemberFQNGateSchema:     stubDef(t, ctx, "CatalogMemberFQNGate"),
		TraitOptionalGateSchema: stubDef(t, ctx, "TraitOptionalGate"),
		Registry:                registry,
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

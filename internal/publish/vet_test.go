package publish

import (
	"context"
	"testing"

	"cuelang.org/go/cue/cuecontext"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runVetChecks runs VetChecks on files with the stub schema and returns the
// plan and the vetted module. None of the fixtures declare debugValues — the
// checks must report regardless.
func runVetChecks(t *testing.T, files map[string]string) (*Plan, *module.Module) {
	t.Helper()
	dir := writeTree(t, files)
	ctx := cuecontext.New()
	p, mod, err := VetChecks(context.Background(), Options{
		Dir:            dir,
		Kind:           KindModule,
		Context:        ctx,
		Kernel:         kernel.New(),
		IdentitySchema: stubSchema(t, ctx),
	})
	require.NoError(t, err)
	require.NotNil(t, mod)
	require.True(t, mod.Package.Exists())
	return p, mod
}

func TestVetChecks_CleanModulePasses(t *testing.T) {
	p, mod := runVetChecks(t, moduleFiles())
	assert.Empty(t, p.Refusals, refusalHeadlines(p))
	assert.Equal(t, "example.com/modules/demo@v1", p.DeclaredPath)
	assert.Equal(t, "demo", p.ModuleName)
	require.NotNil(t, mod.Metadata, "concrete metadata decodes")
	assert.Equal(t, "demo", mod.Metadata.Name)
}

// TestVetChecks_OpenMetadataVersionKeepsName: an open identity Version passes
// the identity check, because --version fills it at publish. An open
// metadata.version then keeps the module's metadata from decoding as a whole,
// yet the plan still carries the authored name for vet's log prefix and the
// module still reaches #config.
func TestVetChecks_OpenMetadataVersionKeepsName(t *testing.T) {
	files := edit(moduleFiles(), "identity/identity.cue", `package identity

ModulePath: "example.com/modules/demo@v1"
Version:    string
`)
	files = edit(files, "module.cue", `package demo

kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v1"
	version:    string
}

#config: replicas: int
`)
	p, mod := runVetChecks(t, files)
	assert.Empty(t, p.Refusals, refusalHeadlines(p))
	assert.Equal(t, "demo", p.ModuleName)
	assert.Nil(t, mod.Metadata, "an open version does not decode")
	assert.True(t, mod.ConfigSchema().Exists())
}

func TestVetChecks_CoordinateDrift(t *testing.T) {
	files := edit(moduleFiles(), "cue.mod/module.cue", `module: "example.com/modules/other@v1"
language: version: "v0.17.0"
source: kind: "self"
`)
	p, _ := runVetChecks(t, files)
	require.Len(t, p.Refusals, 1, refusalHeadlines(p))
	r := p.Refusals[0]
	assert.Contains(t, r.Headline, "disagrees with itself about where it lives")
	// The same aligned two-value form publish uses: both values, both files.
	assert.Contains(t, r.Details(), "example.com/modules/demo@v1")
	assert.Contains(t, r.Details(), "example.com/modules/other@v1")
}

func TestVetChecks_DerivationDrift(t *testing.T) {
	files := edit(moduleFiles(), "module.cue", `package demo

kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v1"
	version:    "1.0.0"
}
`)
	p, _ := runVetChecks(t, files)
	require.Len(t, p.Refusals, 1, refusalHeadlines(p))
	assert.Contains(t, p.Refusals[0].Headline, "states a version its identity package does not")
	assert.Contains(t, p.Refusals[0].Action, "version: id.Version")
}

func TestVetChecks_NonConformantIdentity(t *testing.T) {
	files := edit(moduleFiles(), "identity/identity.cue", `package identity

ModulePath:     "example.com/modules/demo@v1"
CatalogVersion: "1.2.0"
`)
	p, _ := runVetChecks(t, files)
	require.NotEmpty(t, p.Refusals)
	r := p.Refusals[0]
	assert.Contains(t, r.Headline, "#IdentityPackage")
	require.Error(t, r.Err)
	assert.Contains(t, r.Err.Error(), "CatalogVersion")
}

// TestVetChecks_OpenVersionIsNotPoliced: an open Version is a valid authoring
// state; vet does not enforce concreteness — publish does.
func TestVetChecks_OpenVersionIsNotPoliced(t *testing.T) {
	files := edit(moduleFiles(), "identity/identity.cue", `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

ModulePath: "example.com/modules/demo@v1"
Version:    #VersionType
`)
	files = edit(files, "module.cue", `package demo

kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v1"
}
`)
	p, _ := runVetChecks(t, files)
	assert.Empty(t, p.Refusals, refusalHeadlines(p))
}

// TestVetChecks_VersionMajorSkew: the evaluable half (enhancement 0011) at vet — the declared
// version's major must name the path's.
func TestVetChecks_VersionMajorSkew(t *testing.T) {
	files := edit(moduleFiles(), "identity/identity.cue", `package identity

ModulePath: "example.com/modules/demo@v1"
Version:    "2.0.0"
`)
	files = edit(files, "module.cue", `package demo

kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v1"
	version:    "2.0.0"
}
`)
	p, _ := runVetChecks(t, files)
	assert.Contains(t, refusalHeadlines(p), "within the major")
}

func TestVetChecks_KernelLoad(t *testing.T) {
	p, _ := runVetChecks(t, defaultedIdentityFiles())
	require.Len(t, p.Refusals, 1, refusalHeadlines(p))
	assert.Contains(t, p.Refusals[0].Headline, "the kernel would refuse to load this module")
}

// v0ModuleFiles is moduleFiles under a @v0 module path, with the given
// identity Version declaration and no metadata.version: the identity check
// is what these cases are about.
func v0ModuleFiles(versionDecl string) map[string]string {
	identity := `package identity

ModulePath: "example.com/modules/demo@v0"
`
	if versionDecl != "" {
		identity += "Version:    " + versionDecl + "\n"
	}
	return map[string]string{
		"cue.mod/module.cue": `module: "example.com/modules/demo@v0"
language: version: "v0.17.0"
source: kind: "self"
`,
		"identity/identity.cue": identity,
		"module.cue": `package demo

kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v0"
}
`,
	}
}

// TestVetChecks_OpenVersionUnderV0Passes: an open Version under a @v0 path
// passes vet's identity checks, with the stub's VersionMajor derivation in
// place.
func TestVetChecks_OpenVersionUnderV0Passes(t *testing.T) {
	p, _ := runVetChecks(t, v0ModuleFiles("string"))
	assert.Empty(t, p.Refusals, refusalHeadlines(p))
	ver := p.identityField("Version")
	require.NotNil(t, ver)
	assert.Equal(t, StateOpen, ver.State)
}

// TestVetChecks_AbsentVersionFails: an identity package that declares no
// Version fails the identity check, naming the missing field and the file.
func TestVetChecks_AbsentVersionFails(t *testing.T) {
	p, _ := runVetChecks(t, v0ModuleFiles(""))
	require.NotEmpty(t, p.Refusals)
	r := refusalNaming(p, "Version")
	require.NotNil(t, r, "no refusal names the missing Version:\n%s", allRefusalText(p))
	assert.Contains(t, refusalText(*r), "identity/identity.cue")
}

// TestVetChecks_MalformedVersionFails: a Version that is not a version fails
// the identity check with CUE's diagnostic.
func TestVetChecks_MalformedVersionFails(t *testing.T) {
	p, _ := runVetChecks(t, v0ModuleFiles(`"abc"`))
	r := conformanceRefusal(p)
	require.NotNil(t, r, refusalHeadlines(p))
	require.Error(t, r.Err, "the diagnostic is CUE's")
	assert.Contains(t, r.Err.Error(), "abc")
}

// TestVetChecks_SkewedVersionFails: Version "1.0.0" under a @v0 path fails the
// identity check with CUE's diagnostic, which names both majors.
func TestVetChecks_SkewedVersionFails(t *testing.T) {
	p, _ := runVetChecks(t, v0ModuleFiles(`"1.0.0"`))
	r := conformanceRefusal(p)
	require.NotNil(t, r, refusalHeadlines(p))
	require.Error(t, r.Err, "the diagnostic is CUE's")
	assert.Contains(t, r.Err.Error(), `"v0"`)
	assert.Contains(t, r.Err.Error(), `"v1"`)
}

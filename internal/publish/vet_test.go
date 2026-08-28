package publish

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runVetChecks runs VetChecks on files with the stub schema. None of the
// fixtures declare debugValues — the checks must report regardless.
func runVetChecks(t *testing.T, files map[string]string) *Plan {
	t.Helper()
	dir := writeTree(t, files)
	ctx := cuecontext.New()
	p, root, err := VetChecks(Options{
		Dir:            dir,
		Kind:           KindModule,
		Context:        ctx,
		IdentitySchema: stubSchema(t, ctx),
	})
	require.NoError(t, err)
	require.True(t, root.Exists())
	return p
}

func TestVetChecks_CleanModulePasses(t *testing.T) {
	p := runVetChecks(t, moduleFiles())
	assert.Empty(t, p.Refusals, refusalHeadlines(p))
	assert.Equal(t, "example.com/modules/demo@v1", p.DeclaredPath)
}

func TestVetChecks_CoordinateDrift(t *testing.T) {
	files := edit(moduleFiles(), "cue.mod/module.cue", `module: "example.com/modules/other@v1"
language: version: "v0.17.0"
source: kind: "self"
`)
	p := runVetChecks(t, files)
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
	p := runVetChecks(t, files)
	require.Len(t, p.Refusals, 1, refusalHeadlines(p))
	assert.Contains(t, p.Refusals[0].Headline, "states a version its identity package does not")
	assert.Contains(t, p.Refusals[0].Action, "version: id.Version")
}

func TestVetChecks_NonConformantIdentity(t *testing.T) {
	files := edit(moduleFiles(), "identity/identity.cue", `package identity

ModulePath:     "example.com/modules/demo@v1"
CatalogVersion: "1.2.0"
`)
	p := runVetChecks(t, files)
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
	p := runVetChecks(t, files)
	assert.Empty(t, p.Refusals, refusalHeadlines(p))
}

// TestVetChecks_VersionMajorSkew: D18's evaluable half at vet — the declared
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
	p := runVetChecks(t, files)
	assert.Contains(t, refusalHeadlines(p), "within the major")
}

func TestVetChecks_KernelLoad(t *testing.T) {
	p := runVetChecks(t, defaultedIdentityFiles())
	require.Len(t, p.Refusals, 1, refusalHeadlines(p))
	assert.Contains(t, p.Refusals[0].Headline, "the kernel would refuse to load this module")
}

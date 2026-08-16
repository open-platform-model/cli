package publish

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRun_CleanModule_GO pins the clean row of the acceptance matrix: a
// conformant artifact with concrete identity resolves every coordinate from
// its own declarations and passes every gate.
func TestRun_CleanModule_GO(t *testing.T) {
	p := runFixture(t, KindModule, moduleFiles())

	require.Empty(t, p.Refusals, refusalHeadlines(p))
	assert.True(t, p.Go())
	assert.Equal(t, "example.com/modules/demo@v1", p.DeclaredPath)
	assert.Equal(t, "example.com/modules/demo", p.RegistryRepo)
	assert.Equal(t, "v1", p.Major)
	assert.Equal(t, "v1.2.0", p.Tag)
	assert.Equal(t, "the artifact's own Version", p.TagSource)
	assert.Empty(t, p.FillVersion)

	ver := p.identityField("Version")
	require.NotNil(t, ver)
	assert.Equal(t, StateConcrete, ver.State)
	assert.Equal(t, "1.2.0", ver.Value)
	assert.False(t, ver.Defaulted)
	assert.Contains(t, ver.Pos, "identity/identity.cue:")
}

func TestRun_CleanCatalog_GO(t *testing.T) {
	p := runFixture(t, KindCatalog, catalogFiles())
	require.Empty(t, p.Refusals, refusalHeadlines(p))
	assert.Equal(t, "example.com/catalogs/demo@v1", p.DeclaredPath)
	assert.Equal(t, "v1.2.0", p.Tag)
}

// TestRun_DefaultedVersionIsConcrete pins the defaults-are-concrete decision:
// `#VersionType | *"1.2.0"` publishes at the default — the committed,
// release-automation-owned declaration.
func TestRun_DefaultedVersionIsConcrete(t *testing.T) {
	files := edit(moduleFiles(), "identity/identity.cue", `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"

ModulePath: "example.com/modules/demo@v1"
Version:    #VersionType | *"1.2.0"
`)
	p := runFixture(t, KindModule, files)

	require.Empty(t, p.Refusals, refusalHeadlines(p))
	ver := p.identityField("Version")
	assert.Equal(t, StateConcrete, ver.State)
	assert.Equal(t, "1.2.0", ver.Value)
	assert.True(t, ver.Defaulted)
	assert.Equal(t, "v1.2.0", p.Tag)
	assert.Contains(t, p.TagSource, "(default)")
}

// TestRun_VersionStateMatrix covers D3/D12's assert-vs-fill semantics.
func TestRun_VersionStateMatrix(t *testing.T) {
	openIdentity := `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"

ModulePath: "example.com/modules/demo@v1"
Version:    #VersionType
`

	t.Run("concrete plus agreeing flag is a no-op assert", func(t *testing.T) {
		p := runFixture(t, KindModule, moduleFiles(), func(o *Options) { o.VersionFlag = "1.2.0" })
		require.Empty(t, p.Refusals, refusalHeadlines(p))
		assert.Equal(t, "v1.2.0", p.Tag)
		assert.Empty(t, p.FillVersion)
	})

	t.Run("concrete plus disagreeing flag refuses, never overwrites", func(t *testing.T) {
		p := runFixture(t, KindModule, moduleFiles(), func(o *Options) { o.VersionFlag = "1.2.1" })
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		r := p.Refusals[0]
		assert.Contains(t, r.Headline, "--version disagrees with the version this artifact declares")
		assert.Contains(t, r.Details(), "1.2.1")
		assert.Contains(t, r.Details(), "1.2.0")
		assert.Contains(t, r.Action, "opm module version set 1.2.1")
	})

	t.Run("open plus flag fills and records the working-tree write", func(t *testing.T) {
		files := edit(moduleFiles(), "identity/identity.cue", openIdentity)
		// metadata.version's literal would drift from the filled value.
		files = edit(files, "module.cue", `package demo

metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v1"
}
`)
		p := runFixture(t, KindModule, files, func(o *Options) { o.VersionFlag = "1.3.0" })
		require.Empty(t, p.Refusals, refusalHeadlines(p))
		assert.Equal(t, "v1.3.0", p.Tag)
		assert.Equal(t, "1.3.0", p.FillVersion)
		ver := p.identityField("Version")
		assert.True(t, ver.Filled)
		assert.Equal(t, StateOpen, ver.State)
	})

	t.Run("open without flag refuses with the supply actions", func(t *testing.T) {
		files := edit(moduleFiles(), "identity/identity.cue", openIdentity)
		p := runFixture(t, KindModule, files)
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		r := p.Refusals[0]
		assert.Contains(t, r.Headline, "declares an identity field that has no value")
		assert.Contains(t, r.Details(), "identity/identity.cue:")
		assert.Contains(t, r.Action, "opm module version set")
		assert.Contains(t, r.Action, "opm module publish --version")
	})

	t.Run("absent Version yields exactly one refusal, from unification", func(t *testing.T) {
		files := edit(moduleFiles(), "identity/identity.cue", `package identity

ModulePath: "example.com/modules/demo@v1"
`)
		p := runFixture(t, KindModule, files)
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		r := p.Refusals[0]
		assert.Contains(t, r.Headline, "#IdentityPackage")
		require.Error(t, r.Err)
		assert.Contains(t, r.Err.Error(), "Version")
		// --version does not apply to an absent field: still one refusal.
		p2 := runFixture(t, KindModule, files, func(o *Options) { o.VersionFlag = "1.2.0" })
		assert.Len(t, p2.Refusals, 1, refusalHeadlines(p2))
	})
}

// TestRun_IdentityConformance covers refusal 10: CUE's own diagnostic, never
// a hand-rolled comparison.
func TestRun_IdentityConformance(t *testing.T) {
	t.Run("conformant identity passes", func(t *testing.T) {
		p := runFixture(t, KindModule, moduleFiles())
		assert.Empty(t, p.Refusals, refusalHeadlines(p))
	})

	t.Run("renamed field refused by closedness", func(t *testing.T) {
		files := edit(moduleFiles(), "identity/identity.cue", `package identity

ModulePath:     "example.com/modules/demo@v1"
CatalogVersion: "1.2.0"
`)
		p := runFixture(t, KindModule, files)
		require.NotEmpty(t, p.Refusals)
		found := false
		for _, r := range p.Refusals {
			if r.Err != nil && r.Headline == "the identity package does not conform to core's #IdentityPackage" {
				assert.Contains(t, r.Err.Error(), "CatalogVersion")
				found = true
			}
		}
		assert.True(t, found, refusalHeadlines(p))
	})

	t.Run("missing required field refused by unification", func(t *testing.T) {
		files := edit(moduleFiles(), "identity/identity.cue", `package identity

Version: "1.2.0"
`)
		p := runFixture(t, KindModule, files)
		require.NotEmpty(t, p.Refusals)
		r := p.Refusals[0]
		require.Error(t, r.Err)
		assert.Contains(t, r.Err.Error(), "ModulePath")
	})

	t.Run("absent identity package surfaces the load error", func(t *testing.T) {
		files := moduleFiles()
		delete(files, "identity/identity.cue")
		p := runFixture(t, KindModule, files)
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		assert.Contains(t, p.Refusals[0].Headline, "identity package does not load")
		assert.Error(t, p.Refusals[0].Err)
	})
}

// TestConformIdentity_IncompleteWordingCanary pins the CUE SDK error wording
// conformIdentity's filter depends on: openness must keep reading as
// "incomplete value" so it stays D4's gate rather than a conformance failure.
// If a cuelang.org/go bump rewords the diagnostic, this fails in CI instead
// of publish silently misclassifying refusals.
func TestConformIdentity_IncompleteWordingCanary(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`x: string`).LookupPath(cue.ParsePath("x"))
	err := v.Validate(cue.Concrete(true))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "incomplete value",
		"cuelang.org/go changed its incompleteness wording — update conformIdentity's filter in identity.go")
}

// TestRun_LoadFailureShortCircuits: a tree that does not load surfaces the
// CUE error verbatim and reports nothing identity-derived.
func TestRun_LoadFailureShortCircuits(t *testing.T) {
	files := edit(moduleFiles(), "module.cue", `package demo

metadata: name: "demo"
metadata: name: "other"
`)
	p := runFixture(t, KindModule, files)
	require.Len(t, p.Refusals, 1, refusalHeadlines(p))
	r := p.Refusals[0]
	assert.Contains(t, r.Headline, "does not load")
	require.Error(t, r.Err)
	assert.Empty(t, p.DeclaredPath)
	assert.Empty(t, p.Identity)
}

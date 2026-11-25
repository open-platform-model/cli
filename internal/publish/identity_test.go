package publish

import (
	"strings"
	"testing"

	cueerrors "cuelang.org/go/cue/errors"
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

// TestRun_VersionStateMatrix covers the assert-vs-fill semantics of enhancement 0011.
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

kind: "Module"
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
		text := refusalText(p.Refusals[0])
		assert.Contains(t, text, "Version")
		assert.Contains(t, text, "identity/identity.cue")
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
		assert.Contains(t, refusalText(p.Refusals[0]), "ModulePath")
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

// refusalText is all a refusal shows: its headline, its details, and CUE's
// diagnostic with the positions the funnel prints.
func refusalText(r Refusal) string {
	text := r.Headline + "\n" + r.Details()
	if r.Err != nil {
		text += "\n" + cueerrors.Details(r.Err, nil)
	}
	return text
}

// allRefusalText joins refusalText over every refusal of p.
func allRefusalText(p *Plan) string {
	var b strings.Builder
	for _, r := range p.Refusals {
		b.WriteString(refusalText(r))
		b.WriteString("\n")
	}
	return b.String()
}

// refusalNaming returns the first refusal of p whose text names s, or nil.
func refusalNaming(p *Plan, s string) *Refusal {
	for i := range p.Refusals {
		if strings.Contains(refusalText(p.Refusals[i]), s) {
			return &p.Refusals[i]
		}
	}
	return nil
}

// conformanceRefusal returns the refusal of the identity check that carries
// CUE's diagnostic, or nil.
func conformanceRefusal(p *Plan) *Refusal {
	for i := range p.Refusals {
		if p.Refusals[i].Err != nil && strings.Contains(p.Refusals[i].Headline, "#IdentityPackage") {
			return &p.Refusals[i]
		}
	}
	return nil
}

// TestRun_FillThenCheck: --version fills an open Version before the identity
// check, which then checks the identity as filled, in the dry run.
func TestRun_FillThenCheck(t *testing.T) {
	t.Run("a fill within the path major passes conformance", func(t *testing.T) {
		p := runFixture(t, KindModule, v0ModuleFiles("string"), func(o *Options) { o.VersionFlag = "0.1.0" })
		require.Empty(t, p.Refusals, refusalHeadlines(p))
		assert.True(t, p.Go())
		ver := p.identityField("Version")
		require.NotNil(t, ver)
		assert.Equal(t, StateOpen, ver.State)
		assert.True(t, ver.Filled)
		assert.Equal(t, "0.1.0", ver.Value)
		assert.Equal(t, "0.1.0", p.FillVersion)
		assert.Equal(t, "v0.1.0", p.Tag)
		assert.Equal(t, "--version (fills the open Version)", p.TagSource)
	})

	t.Run("a fill with another major is refused, naming both majors", func(t *testing.T) {
		p := runFixture(t, KindModule, v0ModuleFiles("string"), func(o *Options) { o.VersionFlag = "1.0.0" })
		require.NotEmpty(t, p.Refusals)
		assert.False(t, p.Go(), "nothing is pushed")
		text := allRefusalText(p)
		assert.Contains(t, text, "v1")
		assert.Contains(t, text, "v0")
		// The identity check sees the filled identity: when it refuses, it
		// refuses the major skew, not the open field.
		if r := conformanceRefusal(p); r != nil {
			assert.Contains(t, r.Err.Error(), `"v1"`, "the identity check did not see the fill")
			assert.Contains(t, r.Err.Error(), `"v0"`, "the identity check did not see the fill")
		}
	})

	t.Run("an open Version without the flag is only the open-field refusal", func(t *testing.T) {
		p := runFixture(t, KindModule, v0ModuleFiles("string"))
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		assert.Contains(t, p.Refusals[0].Headline, "declares an identity field that has no value")
	})

	t.Run("a malformed Version is refused with CUE's diagnostic", func(t *testing.T) {
		p := runFixture(t, KindModule, v0ModuleFiles(`"abc"`))
		r := conformanceRefusal(p)
		require.NotNil(t, r, refusalHeadlines(p))
		assert.Contains(t, r.Err.Error(), "abc")
	})

	t.Run("a skewed Version is refused with CUE's diagnostic", func(t *testing.T) {
		p := runFixture(t, KindModule, v0ModuleFiles(`"1.0.0"`))
		r := conformanceRefusal(p)
		require.NotNil(t, r, refusalHeadlines(p))
		assert.Contains(t, r.Err.Error(), `"v0"`)
		assert.Contains(t, r.Err.Error(), `"v1"`)
	})

	t.Run("an absent Version is refused, naming the field and its file", func(t *testing.T) {
		p := runFixture(t, KindModule, v0ModuleFiles(""))
		r := refusalNaming(p, "Version")
		require.NotNil(t, r, allRefusalText(p))
		assert.Contains(t, refusalText(*r), "identity/identity.cue")
	})
}

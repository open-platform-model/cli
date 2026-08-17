package publish

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withDeclaredPath rewrites a module fixture's three path statements to the
// given declared path (major included), declaring a version whose major
// matches the path's.
func withDeclaredPath(files map[string]string, path string) map[string]string {
	version := versionForPath(path)
	out := edit(files, "cue.mod/module.cue", fmt.Sprintf(`module: %q
language: version: "v0.17.0"
source: kind: "self"
`, path))
	out = edit(out, "identity/identity.cue", fmt.Sprintf(`package identity

ModulePath: %q
Version:    %q
`, path, version))
	out = edit(out, "module.cue", fmt.Sprintf(`package demo

metadata: {
	name:       "demo"
	modulePath: %q
	version:    %q
}
`, path, version))
	return out
}

func TestGate_NoCueMod(t *testing.T) {
	files := moduleFiles()
	delete(files, "cue.mod/module.cue")
	p := runFixture(t, KindModule, files)

	require.Len(t, p.Refusals, 1, refusalHeadlines(p))
	r := p.Refusals[0]
	assert.Contains(t, r.Headline, "no cue.mod/module.cue")
	assert.Contains(t, r.Headline, "this is not a CUE module")
	// The identity package still loads on its own, so the action names the
	// declared path rather than a placeholder.
	assert.Contains(t, r.Action, "opm mod init example.com/modules/demo@v1")
}

func TestGate_CueModAgreement(t *testing.T) {
	files := edit(moduleFiles(), "cue.mod/module.cue", `module: "example.com/modules/other@v1"
language: version: "v0.17.0"
source: kind: "self"
`)
	p := runFixture(t, KindModule, files)

	require.Len(t, p.Refusals, 1, refusalHeadlines(p))
	r := p.Refusals[0]
	assert.Contains(t, r.Headline, "disagrees with itself about where it lives")
	details := r.Details()
	assert.Contains(t, details, "example.com/modules/demo@v1")
	assert.Contains(t, details, "example.com/modules/other@v1")
	assert.Contains(t, details, "identity/identity.cue:")
	assert.Contains(t, details, "cue.mod/module.cue:")
	assert.Contains(t, r.Action, "publish will not choose for you")
}

func TestGate_SourceSelf(t *testing.T) {
	t.Run("absent source refused", func(t *testing.T) {
		files := edit(moduleFiles(), "cue.mod/module.cue", `module: "example.com/modules/demo@v1"
language: version: "v0.17.0"
`)
		p := runFixture(t, KindModule, files)
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		assert.Contains(t, p.Refusals[0].Headline, `source: {kind: "self"}`)
		assert.Contains(t, p.Refusals[0].Action, "cue mod edit --source self")
	})

	t.Run("other kind refused with the value named", func(t *testing.T) {
		files := edit(moduleFiles(), "cue.mod/module.cue", `module: "example.com/modules/demo@v1"
language: version: "v0.17.0"
source: kind: "git"
`)
		p := runFixture(t, KindModule, files)
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		assert.Contains(t, p.Refusals[0].Details(), `{kind: "git"}`)
	})
}

func TestGate_Derivation(t *testing.T) {
	t.Run("metadata.version literal drift is msg 7", func(t *testing.T) {
		files := edit(moduleFiles(), "module.cue", `package demo

metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v1"
	version:    "1.1.0"
}
`)
		p := runFixture(t, KindModule, files)
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		r := p.Refusals[0]
		assert.Contains(t, r.Headline, "states a version its identity package does not")
		details := r.Details()
		assert.Contains(t, details, "metadata.version")
		assert.Contains(t, details, "1.1.0")
		assert.Contains(t, details, "id.Version")
		assert.Contains(t, details, "1.2.0")
		assert.Contains(t, r.Action, "version: id.Version")
	})

	t.Run("metadata.modulePath drift caught too", func(t *testing.T) {
		files := edit(moduleFiles(), "module.cue", `package demo

metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo-x@v1"
	version:    "1.2.0"
}
`)
		p := runFixture(t, KindModule, files)
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		assert.Contains(t, p.Refusals[0].Action, "modulePath: id.ModulePath")
	})

	t.Run("filled version compared against the fill", func(t *testing.T) {
		files := edit(moduleFiles(), "identity/identity.cue", `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

ModulePath: "example.com/modules/demo@v1"
Version:    #VersionType
`)
		// metadata.version keeps a literal that will disagree once --version
		// fills the open field: published bytes would be inconsistent, so it
		// is caught now.
		p := runFixture(t, KindModule, files, func(o *Options) { o.VersionFlag = "1.3.0" })
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		assert.Contains(t, p.Refusals[0].Headline, "states a version its identity package does not")
	})
}

func TestGate_TagMajor(t *testing.T) {
	files := edit(moduleFiles(), "identity/identity.cue", `package identity

ModulePath: "example.com/modules/demo@v2"
Version:    "1.2.0"
`)
	files = edit(files, "cue.mod/module.cue", `module: "example.com/modules/demo@v2"
language: version: "v0.17.0"
source: kind: "self"
`)
	files = edit(files, "module.cue", `package demo

metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v2"
	version:    "1.2.0"
}
`)
	p := runFixture(t, KindModule, files)

	require.NotEmpty(t, p.Refusals)
	var tagRefusal *Refusal
	for i := range p.Refusals {
		if strings.Contains(p.Refusals[i].Headline, "within the major") {
			tagRefusal = &p.Refusals[i]
		}
	}
	require.NotNil(t, tagRefusal, refusalHeadlines(p))
	details := tagRefusal.Details()
	assert.Contains(t, details, "v1.2.0")
	assert.Contains(t, details, "v2")
}

// TestGate_Namespace pins the four passing cases from 0011's
// schemas/target.cue — the constraint D13 exists to keep from growing until
// it refuses a third party's own domain — and the measured must-fail shapes.
func TestGate_Namespace(t *testing.T) {
	passing := []struct {
		name string
		kind Kind
		path string
	}{
		{"vanity domain is unconstrained", KindCatalog, "example.com/k8up@v1"},
		{"testing domain is a separate domain, not policed", KindModule, "testing.opmodel.dev/modules/web_app@v1"},
		{"community module with owner segment", KindModule, "community.opmodel.dev/m/acme/media_server@v2"},
		{"opmodel.dev/core is the fixed path outside the kind pattern", KindCatalog, "opmodel.dev/core@v1"},
		{"reserved templates segment is module-kind (D25)", KindModule, "opmodel.dev/templates/standard@v1"},
	}
	for _, tt := range passing {
		t.Run(tt.name, func(t *testing.T) {
			files := withDeclaredPath(moduleFiles(), tt.path)
			p := runFixture(t, tt.kind, files)
			assert.Empty(t, p.Refusals, refusalHeadlines(p))
		})
	}

	refused := []struct {
		name string
		kind Kind
		path string
		want string
	}{
		{"abandoned owner-scoped first-party shape", KindModule, "opmodel.dev/m/acme/postgres@v1", "does not fit the namespace"},
		{"unreserved releases segment", KindModule, "opmodel.dev/releases/prod@v1", "does not fit the namespace"},
		{"nested path under a valid kind segment", KindModule, "opmodel.dev/modules/test/hello_web@v1", "does not fit the namespace"},
		{"module published to a catalogs path", KindModule, "opmodel.dev/catalogs/opm@v1", "cannot publish under the catalogs/ segment"},
		{"catalog published to a modules path", KindCatalog, "opmodel.dev/modules/demo@v1", "cannot publish under the modules/ segment"},
		{"nested path under the templates segment (D25)", KindModule, "opmodel.dev/templates/x/y@v1", "does not fit the namespace"},
		{"catalog published to a templates path (D25)", KindCatalog, "opmodel.dev/templates/opm@v1", "cannot publish under the templates/ segment"},
	}
	for _, tt := range refused {
		t.Run(tt.name, func(t *testing.T) {
			files := withDeclaredPath(moduleFiles(), tt.path)
			p := runFixture(t, tt.kind, files)
			assert.Contains(t, refusalHeadlines(p), tt.want)
		})
	}
}

// versionForPath returns a version whose major matches the path's, so the
// namespace pins are not polluted by tag-major refusals.
func versionForPath(path string) string {
	if strings.HasSuffix(path, "@v2") {
		return "2.4.1"
	}
	return "1.2.0"
}

func TestGate_PackageName(t *testing.T) {
	misnamed := edit(moduleFiles(), "module.cue", `package demo_db

metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v1"
	version:    "1.2.0"
}
`)

	t.Run("module package must bind to metadata.name", func(t *testing.T) {
		p := runFixture(t, KindModule, misnamed)
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		r := p.Refusals[0]
		assert.Contains(t, r.Headline, "package does not bind to its name")
		details := r.Details()
		assert.Contains(t, details, "demo_db")
		assert.Contains(t, details, "module.cue:1")
		assert.Contains(t, details, "via metadata.name")
		assert.Contains(t, r.Action, "package demo")
	})

	t.Run("catalog entry point skips the gate", func(t *testing.T) {
		// catalogFiles' root package (democat) already differs from
		// metadata.name (demo); the clean-catalog test passing is the same
		// pin, restated here for the contrast.
		p := runFixture(t, KindCatalog, catalogFiles())
		assert.Empty(t, p.Refusals, refusalHeadlines(p))
	})
}

// TestGate_Override covers all four presence/flag states for both kinds
// (D6). The flag waives the gate for modules only, and changes nothing else.
func TestGate_Override(t *testing.T) {
	// The override entry names a dep that is pinned but never imported, so
	// the tree still loads without resolving it.
	withOverride := func(files map[string]string, modPath string) map[string]string {
		out := edit(files, "cue.mod/module.cue", fmt.Sprintf(`module: %q
language: version: "v0.17.0"
source: kind: "self"
deps: "example.com/dep@v1": v: "v1.4.0"
`, modPath))
		return edit(out, "cue.mod/local-module.cue", `deps: "example.com/dep@v1": replaceWith: "../dep"
`)
	}

	t.Run("module with override refused, escape hatch offered by name", func(t *testing.T) {
		p := runFixture(t, KindModule, withOverride(moduleFiles(), "example.com/modules/demo@v1"))
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		r := p.Refusals[0]
		assert.Contains(t, r.Headline, "local-module.cue is present")
		details := r.Details()
		assert.Contains(t, details, "example.com/dep@v1")
		assert.Contains(t, details, "../dep")
		assert.Contains(t, details, "would resolve to v1.4.0 when published")
		assert.Contains(t, r.Action, "--skip-override-check")
		assert.True(t, p.OverridePresent)
	})

	t.Run("module with override and flag proceeds, gate waived only", func(t *testing.T) {
		p := runFixture(t, KindModule, withOverride(moduleFiles(), "example.com/modules/demo@v1"),
			func(o *Options) { o.SkipOverrideCheck = true })
		assert.Empty(t, p.Refusals, refusalHeadlines(p))
		assert.True(t, p.OverridePresent)
		assert.True(t, p.OverrideWaived)
		// The replacements are still reported — ignored, not honored.
		require.Len(t, p.Replacements, 1)
		assert.Equal(t, "v1.4.0", p.Replacements[0].PublishedVersion)
	})

	t.Run("catalog with override refused", func(t *testing.T) {
		p := runFixture(t, KindCatalog, withOverride(catalogFiles(), "example.com/catalogs/demo@v1"))
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		assert.Contains(t, p.Refusals[0].Headline, "a catalog cannot be published")
		assert.NotContains(t, p.Refusals[0].Action, "--skip-override-check")
	})

	t.Run("catalog with override and flag still refused", func(t *testing.T) {
		p := runFixture(t, KindCatalog, withOverride(catalogFiles(), "example.com/catalogs/demo@v1"),
			func(o *Options) { o.SkipOverrideCheck = true })
		require.Len(t, p.Refusals, 1, refusalHeadlines(p))
		assert.False(t, p.OverrideWaived)
	})

	t.Run("clean trees pass for both kinds", func(t *testing.T) {
		assert.Empty(t, runFixture(t, KindModule, moduleFiles()).Refusals)
		assert.Empty(t, runFixture(t, KindCatalog, catalogFiles()).Refusals)
	})
}

// TestRun_RefusalsAccumulate: every evaluable gate runs and one invocation
// reports every defect — one pass fixes everything.
func TestRun_RefusalsAccumulate(t *testing.T) {
	files := edit(moduleFiles(), "cue.mod/module.cue", `module: "example.com/modules/other@v1"
language: version: "v0.17.0"
deps: "example.com/dep@v1": v: "v1.4.0"
`)
	files = edit(files, "cue.mod/local-module.cue", `deps: "example.com/dep@v1": replaceWith: "../dep"
`)
	p := runFixture(t, KindModule, files)

	headlines := refusalHeadlines(p)
	assert.Contains(t, headlines, "disagrees with itself")
	assert.Contains(t, headlines, "source")
	assert.Contains(t, headlines, "local-module.cue is present")
	assert.Len(t, p.Refusals, 3, headlines)
}

func TestPlan_Render(t *testing.T) {
	t.Run("GO plan shows every resolved coordinate", func(t *testing.T) {
		p := runFixture(t, KindModule, moduleFiles())
		out := p.Render()
		assert.Contains(t, out, "kind            module")
		assert.Contains(t, out, "declaredPath    example.com/modules/demo@v1")
		assert.Contains(t, out, "cueModPath      example.com/modules/demo@v1")
		assert.Contains(t, out, "registryRepo    example.com/modules/demo")
		assert.Contains(t, out, "major           v1")
		assert.Contains(t, out, "tag             v1.2.0")
		assert.Contains(t, out, "tag from        the artifact's own Version")
		assert.Contains(t, out, "ModulePath")
		assert.Contains(t, out, "Version")
		assert.Contains(t, out, "local override  false")
		assert.Contains(t, out, "GO — pushing example.com/modules/demo:v1.2.0")
	})

	t.Run("refused plan carries the count, not the details", func(t *testing.T) {
		files := edit(moduleFiles(), "cue.mod/module.cue", `module: "example.com/modules/other@v1"
language: version: "v0.17.0"
source: kind: "self"
`)
		p := runFixture(t, KindModule, files)
		out := p.Render()
		assert.Contains(t, out, "REFUSED — 1 refusal")
		assert.NotContains(t, out, "GO")
	})
}

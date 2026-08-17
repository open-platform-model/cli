package publish

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// coldCUECache points CUE_CACHE_DIR at a fresh directory for this test. The
// cache extracts module files read-only, so t.TempDir's own cleanup would
// fail on them — the cleanup here chmods before removing.
func coldCUECache(t *testing.T) {
	t.Helper()
	dir, err := os.MkdirTemp("", "opm-cue-cache-*")
	require.NoError(t, err)
	t.Setenv("CUE_CACHE_DIR", dir)
	t.Cleanup(func() {
		_ = filepath.WalkDir(dir, func(p string, _ fs.DirEntry, err error) error {
			if err == nil {
				_ = os.Chmod(p, 0o700)
			}
			return nil
		})
		_ = os.RemoveAll(dir)
	})
}

// The multi-build tests below publish real builds into the in-process
// registry and then gate the next one. Every test that fetches published
// history pins CUE_CACHE_DIR to a fresh temp dir: the on-disk module cache is
// keyed by path@version alone, and every test publishes the same fixture path
// with different bytes — a warm shared cache would serve one test's build to
// another (and, measured, makes dead-registry setups pass by never dialing).

// memberCatalogFilesAt is memberCatalogFiles restamped at another catalog
// version — identity Version, every member's catalogVersion, and the
// transformer's build-keyed fqn move together, as a release does.
func memberCatalogFilesAt(version string) map[string]string {
	files := memberCatalogFiles()
	out := make(map[string]string, len(files))
	for k, v := range files {
		out[k] = strings.ReplaceAll(v, "1.2.0", version)
	}
	return out
}

// pushCatalog publishes a catalog fixture build for real.
func pushCatalog(t *testing.T, registry string, files map[string]string) {
	t.Helper()
	dir := writeTree(t, files)
	var opts Options
	p := runDir(t, KindCatalog, dir, func(o *Options) {
		o.Registry = registry
		opts = *o
	})
	require.True(t, p.Go(), refusalHeadlines(p))
	require.NoError(t, Push(context.Background(), opts, p))
}

// runCatalog computes the plan for a catalog fixture against the registry.
func runCatalog(t *testing.T, registry string, files map[string]string) *Plan {
	t.Helper()
	dir := writeTree(t, files)
	return runDir(t, KindCatalog, dir, func(o *Options) { o.Registry = registry })
}

// refusalDetails joins every refusal's rendered detail block.
func refusalDetails(p *Plan) string {
	var b strings.Builder
	for _, r := range p.Refusals {
		b.WriteString(r.Details())
		b.WriteString("\n")
	}
	return b.String()
}

func TestGateCompat_CleanHistoryPasses(t *testing.T) {
	coldCUECache(t)
	registry := emptyTestRegistry(t)
	pushCatalog(t, registry, memberCatalogFilesAt("1.0.0"))

	p := runCatalog(t, registry, memberCatalogFilesAt("1.2.0"))
	require.True(t, p.Go(), refusalHeadlines(p))
	require.True(t, p.CompatChecked)

	// The four beta/GA contract members compared; the alpha trait was
	// exempted before any registry work; the transformer never entered.
	assert.Equal(t, 4, p.CatalogGates.CompatCompared)
	assert.Equal(t, 1, p.CatalogGates.CompatAlpha)
	assert.Equal(t, 0, p.CatalogGates.CompatNew)
	assert.Equal(t, 0, p.CatalogGates.CompatRefused)
	assert.Contains(t, p.Render(), "4 compared, 0 refused, 1 alpha-exempt, 0 new")
	assert.Contains(t, p.Render(), "GO —")
}

func TestGateCompat_AdditiveChangeComparesClean(t *testing.T) {
	// A genuinely changed member (so the identical-modulo-provenance fast
	// path cannot fire) whose change is additive: the walk itself must pass
	// it.
	coldCUECache(t)
	registry := emptyTestRegistry(t)
	pushCatalog(t, registry, memberCatalogFilesAt("1.0.0"))

	next := memberCatalogFilesAt("1.2.0")
	next["resources/v1beta1/thing.cue"] = strings.Replace(
		next["resources/v1beta1/thing.cue"], "note?: string", "note?: string\n\t\tadded?: int", 1)
	p := runCatalog(t, registry, next)
	require.True(t, p.Go(), refusalHeadlines(p)+refusalDetails(p))
	assert.Equal(t, 4, p.CatalogGates.CompatCompared)
}

func TestGateCompat_FieldRemovedRefused(t *testing.T) {
	coldCUECache(t)
	registry := emptyTestRegistry(t)
	pushCatalog(t, registry, memberCatalogFilesAt("1.0.0"))

	next := memberCatalogFilesAt("1.2.0")
	next["resources/v1beta1/thing.cue"] = strings.Replace(
		next["resources/v1beta1/thing.cue"], "size: int\n", "", 1)
	p := runCatalog(t, registry, next)
	require.False(t, p.Go())
	require.Equal(t, 1, p.CatalogGates.CompatRefused)

	r := p.Refusals[len(p.Refusals)-1]
	assert.Contains(t, r.Headline, "example.com/catalogs/demo would break a contract it already published")
	details := r.Details()
	assert.Contains(t, details, "#ThingResource")
	assert.Contains(t, details, "v1beta1")
	assert.Contains(t, details, "compared against example.com/catalogs/demo@1.0.0")
	assert.Contains(t, details, "spec.size")
	assert.Contains(t, details, "field removed")
	assert.Contains(t, details, "may gain fields and options, never lose them")
	assert.Contains(t, r.Action, "new apiVersion (v1beta2)")
}

func TestGateCompat_RemoveThenReaddRefused(t *testing.T) {
	// The laundering hole D9's literal rule closes: remove the member in one
	// build, re-add it reshaped in the next. The immediate predecessor does
	// not carry it — the scan probes past the absent package (the negative
	// signal) and finds the older build.
	coldCUECache(t)
	registry := emptyTestRegistry(t)
	pushCatalog(t, registry, memberCatalogFilesAt("1.0.0"))

	removed := memberCatalogFilesAt("1.1.0")
	delete(removed, "resources/v1beta1/thing.cue")
	pushCatalog(t, registry, removed)

	readded := memberCatalogFilesAt("1.2.0")
	readded["resources/v1beta1/thing.cue"] = strings.Replace(
		readded["resources/v1beta1/thing.cue"], "size: int\n", "", 1)
	p := runCatalog(t, registry, readded)
	require.False(t, p.Go(), "an incompatible re-introduction must be refused against the older build")
	assert.Contains(t, refusalDetails(p), "compared against example.com/catalogs/demo@1.0.0")
	assert.Contains(t, refusalDetails(p), "spec.size")
}

func TestGateCompat_PrereleasePredecessorCompared(t *testing.T) {
	// The divergence from a stable-preferring selector (the struck
	// highestStable note): the newest build carrying the member is a
	// prerelease, and the comparison runs against IT — a break only
	// prerelease pinners (D14-blessed) can see is still a break.
	coldCUECache(t)
	registry := emptyTestRegistry(t)
	pushCatalog(t, registry, memberCatalogFilesAt("1.0.0"))

	pre := memberCatalogFilesAt("1.1.0-alpha.1")
	pre["resources/v1beta1/thing.cue"] = strings.Replace(
		pre["resources/v1beta1/thing.cue"], "note?: string", "note?: string\n\t\textra?: string", 1)
	pushCatalog(t, registry, pre)

	// The current build drops extra? — clean against 1.0.0, a removal
	// against 1.1.0-alpha.1.
	p := runCatalog(t, registry, memberCatalogFilesAt("1.2.0"))
	require.False(t, p.Go(), "the prerelease predecessor must be the comparison target")
	details := refusalDetails(p)
	assert.Contains(t, details, "compared against example.com/catalogs/demo@1.1.0-alpha.1")
	assert.Contains(t, details, "spec.extra")
}

func TestGateCompat_NewPackagePasses(t *testing.T) {
	// The apiVersion-bump escape: a member at a key no published build has
	// carried passes trivially, package and all.
	coldCUECache(t)
	registry := emptyTestRegistry(t)
	pushCatalog(t, registry, memberCatalogFilesAt("1.0.0"))

	next := memberCatalogFilesAt("1.2.0")
	next["resources/v2/another.cue"] = `package v2

#AnotherResource: {
	kind: "Resource"
	metadata: {
		modulePath:     "example.com/catalogs/demo/resources/v2"
		name:           "another"
		apiVersion:     "v2"
		catalogVersion: "1.2.0"
		fqn:            "example.com/catalogs/demo/resources/another@v2"
	}
	spec: {
		whole: string
	}
}
`
	p := runCatalog(t, registry, next)
	require.True(t, p.Go(), refusalHeadlines(p))
	assert.Equal(t, 1, p.CatalogGates.CompatNew)
	assert.Equal(t, 4, p.CatalogGates.CompatCompared)
}

func TestGateCompat_ConnectivityAbortRendersNoVerdict(t *testing.T) {
	// A transport failure mid-walk aborts with no partial verdict: the plan
	// stays refusal-free and renders INCOMPLETE, never GO. Cold cache — a
	// warm one would satisfy the fetch without dialing (measured).
	coldCUECache(t)
	dir := writeTree(t, memberCatalogFilesAt("1.2.0"))
	opts := baseOptions(t, dir)
	opts.Registry = "127.0.0.1:1+insecure"

	members, refusals := enumerateMembers(opts, dir)
	require.Empty(t, refusals)
	p := &Plan{
		Kind:              KindCatalog,
		Dir:               dir,
		RegistryRepo:      "example.com/catalogs/demo",
		Major:             "v1",
		Tag:               "v1.2.0",
		RegistryChecked:   true,
		members:           members,
		publishedVersions: []string{"v1.0.0"},
	}

	err := gateCompat(p, opts)
	require.Error(t, err)
	var connErr *ConnectivityError
	require.ErrorAs(t, err, &connErr)

	assert.False(t, p.CompatChecked)
	assert.True(t, p.Go(), "a transport failure is never a refusal")
	assert.Contains(t, p.Render(), "INCOMPLETE")
	assert.NotContains(t, p.Render(), "GO —")
	assert.Contains(t, p.Render(), "compat gate")
	assert.Contains(t, p.Render(), "not completed")
}

func TestGateCompat_FirstPublishAllNew(t *testing.T) {
	// No history at all: every eligible member is new, nothing is fetched,
	// and the walk still completes.
	p := runFixture(t, KindCatalog, memberCatalogFiles())
	require.True(t, p.Go(), refusalHeadlines(p))
	assert.True(t, p.CompatChecked)
	assert.Equal(t, 4, p.CatalogGates.CompatNew)
	assert.Equal(t, 0, p.CatalogGates.CompatCompared)
}

func TestPredecessorVersions_WindowAndOrder(t *testing.T) {
	got := predecessorVersions(
		[]string{"v1.0.0", "v2.0.0", "v1.1.0-alpha.1", "v1.2.0", "v1.1.0", "bogus", "v1.3.0"},
		"v1.2.0", "v1")
	// Strictly below the effective tag, same major, prereleases included,
	// newest first.
	assert.Equal(t, []string{"v1.1.0", "v1.1.0-alpha.1", "v1.0.0"}, got)
}

func TestNextAPIVersion(t *testing.T) {
	assert.Equal(t, "v1beta2", nextAPIVersion("v1beta1"))
	assert.Equal(t, "v1alpha3", nextAPIVersion("v1alpha2"))
	assert.Equal(t, "v2", nextAPIVersion("v1"))
	assert.Equal(t, "<next-apiVersion>", nextAPIVersion("weird"))
}

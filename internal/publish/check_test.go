package publish

import (
	"context"
	"os"
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/mod/module"
	"cuelang.org/go/mod/modzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rawPush publishes a tree through CUE's module machinery directly, gates
// bypassed — what `cue mod publish` does. The check exists precisely for
// builds that arrived this way (D7: publish-side enforcement only protects
// artifacts pushed through OPM tooling).
func rawPush(t *testing.T, registry string, files map[string]string, path, tag string) {
	t.Helper()
	dir := writeTree(t, files)
	client, err := NewRegistryClient(registry)
	require.NoError(t, err)
	mv, err := module.NewVersion(path, tag)
	require.NoError(t, err)
	zipFile, err := os.CreateTemp(t.TempDir(), "raw-*.zip")
	require.NoError(t, err)
	defer zipFile.Close()
	require.NoError(t, modzip.CreateFromDir(zipFile, mv, dir))
	info, err := zipFile.Stat()
	require.NoError(t, err)
	require.NoError(t, client.PutModule(context.Background(), mv, zipFile, info.Size()))
}

func checkFixture(t *testing.T, registry, coordinate string, compat bool) (*CheckReport, error) {
	t.Helper()
	return RegistryCheck(context.Background(), CheckOptions{
		Coordinate: coordinate,
		Compat:     compat,
		Context:    cuecontext.New(),
		Registry:   registry,
	})
}

func TestRegistryCheck_CleanBuild(t *testing.T) {
	coldCUECache(t)
	registry := emptyTestRegistry(t)
	pushCatalog(t, registry, memberCatalogFilesAt("1.2.0"))

	report, err := checkFixture(t, registry, "example.com/catalogs/demo@v1.2.0", false)
	require.NoError(t, err)
	assert.True(t, report.Clean(), report.Render())
	assert.Equal(t, "example.com/catalogs/demo@v1", report.DeclaredPath)
	assert.Equal(t, "1.2.0", report.DeclaredVersion)
	assert.Len(t, report.Members, 6)

	out := report.Render()
	assert.Contains(t, out, "CLEAN")
	assert.Contains(t, out, "resources/v1beta1")
	assert.Contains(t, out, "thing")
	assert.Contains(t, out, "transformers")
}

func TestRegistryCheck_IdentityMismatchFoundOutOfBand(t *testing.T) {
	// A build pushed around the gates: the tree's cue.mod names other@v1 (so
	// the zip is accepted there) while its metadata still declares demo@v1 at
	// a version the tag does not carry. The consumer's check reports both
	// disagreements, values side by side.
	coldCUECache(t)
	registry := emptyTestRegistry(t)

	files := memberCatalogFilesAt("1.9.9")
	files["cue.mod/module.cue"] = strings.ReplaceAll(
		files["cue.mod/module.cue"], "example.com/catalogs/demo@v1", "example.com/catalogs/other@v1")
	rawPush(t, registry, files, "example.com/catalogs/other@v1", "v1.2.0")

	report, err := checkFixture(t, registry, "example.com/catalogs/other@v1.2.0", false)
	require.NoError(t, err)
	require.False(t, report.Clean())

	headlines := ""
	details := ""
	for _, f := range report.Findings {
		headlines += f.Headline + "\n"
		details += f.Details() + "\n"
	}
	assert.Contains(t, headlines, "declares a version other than the tag")
	assert.Contains(t, details, "1.9.9")
	assert.Contains(t, details, "1.2.0")
	assert.Contains(t, headlines, "declares a path other than the coordinate")
	assert.Contains(t, details, "example.com/catalogs/demo@v1")
}

func TestRegistryCheck_CompatAidOverPublishedBuild(t *testing.T) {
	// --compat reports, over a published build, the same violations publish
	// would have refused — here a break that shipped because the build went
	// around the gate.
	coldCUECache(t)
	registry := emptyTestRegistry(t)
	pushCatalog(t, registry, memberCatalogFilesAt("1.0.0"))

	broken := memberCatalogFilesAt("1.2.0")
	broken["resources/v1beta1/thing.cue"] = strings.Replace(
		broken["resources/v1beta1/thing.cue"], "size: int\n", "", 1)
	rawPush(t, registry, broken, "example.com/catalogs/demo@v1", "v1.2.0")

	report, err := checkFixture(t, registry, "example.com/catalogs/demo@v1.2.0", true)
	require.NoError(t, err)
	require.True(t, report.CompatRan)
	require.False(t, report.Clean())
	assert.Equal(t, 1, report.Gates.CompatRefused)

	var details string
	for _, f := range report.Findings {
		details += f.Headline + "\n" + f.Details() + "\n"
	}
	assert.Contains(t, details, "would break a contract it already published")
	assert.Contains(t, details, "compared against example.com/catalogs/demo@1.0.0")
	assert.Contains(t, details, "spec.size")
	assert.Contains(t, report.Render(), "4 compared, 1 violating")
}

func TestRegistryCheck_NotPublished(t *testing.T) {
	coldCUECache(t)
	registry := emptyTestRegistry(t)
	_, err := checkFixture(t, registry, "example.com/catalogs/demo@v9.9.9", false)
	require.ErrorIs(t, err, ErrNotPublished)
}

func TestRegistryCheck_UnreachableRegistryIsConnectivity(t *testing.T) {
	coldCUECache(t)
	_, err := checkFixture(t, "127.0.0.1:1+insecure", "example.com/catalogs/demo@v1.2.0", false)
	var connErr *ConnectivityError
	require.ErrorAs(t, err, &connErr)
}

func TestRegistryCheck_BadCoordinate(t *testing.T) {
	for _, coord := range []string{"example.com/catalogs/demo", "@v1.2.0", "demo@", "demo@vnope"} {
		_, err := checkFixture(t, emptyTestRegistry(t), coord, false)
		require.Error(t, err, coord)
	}
}

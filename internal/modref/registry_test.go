package modref

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testModule is one published version in the in-memory registry: its
// major-qualified module path, version and the core major it depends on
// ("" for no core dependency).
type testModule struct {
	path, version, coreMajor string
}

// testRegistry serves the given module versions from an in-memory registry
// and returns a Source over it whose module cache lives in a temp directory.
func testRegistry(t *testing.T, mods ...testModule) Source {
	t.Helper()
	fsys := fstest.MapFS{}
	for _, m := range mods {
		root, _, _ := strings.Cut(m.path, "@")
		dir := strings.ReplaceAll(root, "/", "_") + "_" + m.version
		modFile := "module: \"" + m.path + "\"\nlanguage: version: \"v0.17.0\"\n"
		if m.coreMajor != "" {
			modFile += "deps: \"opmodel.dev/core@" + m.coreMajor + "\": v: \"" + m.coreMajor + ".0.0\"\n"
		}
		fsys[dir+"/cue.mod/module.cue"] = &fstest.MapFile{Data: []byte(modFile)}
		fsys[dir+"/module.cue"] = &fstest.MapFile{Data: []byte("package m\n")}
	}
	reg, err := modregistrytest.New(fsys, "")
	require.NoError(t, err)
	t.Cleanup(reg.Close)
	return testSource(t, reg.Host()+"+insecure")
}

// testSource builds a Source over registry with an isolated module cache.
func testSource(t *testing.T, registry string) Source {
	t.Helper()
	src, err := modconfig.NewRegistry(&modconfig.Config{
		CUERegistry: registry,
		Env:         []string{"CUE_CACHE_DIR=" + t.TempDir()},
	})
	require.NoError(t, err)
	return src
}

// The resolver lists every major of a module by asking for its major-free
// path; this pins that the in-memory registry answers the way GHCR does.
func TestModuleVersions_MajorFreePathListsEveryMajor(t *testing.T) {
	src := testRegistry(t,
		testModule{"example.com/modules/web_app@v0", "v0.3.0", "v2"},
		testModule{"example.com/modules/web_app@v1", "v1.0.4", "v2"},
	)
	versions, err := src.ModuleVersions(context.Background(), "example.com/modules/web_app")
	require.NoError(t, err)
	assert.Equal(t, []string{"v0.3.0", "v1.0.4"}, versions)
}

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

// TestPins_EqualTheSourcePins checks that the compiled pins equal the source
// pins: the library require in the cli's go.mod, the DefaultSchemaModule line
// of that library's opm/schema/loader.go, and the PinnedOperatorVersion line
// of internal/operator/manifest.go.
func TestPins_EqualTheSourcePins(t *testing.T) {
	got, err := pins()
	require.NoError(t, err)

	lib := goModLibrary(t, "../../go.mod")
	libDir := moduleDir(t, libraryModule)
	core := constValue(t, filepath.Join(libDir, "opm", "schema", "loader.go"), "DefaultSchemaModule", `opmodel\.dev/core@`)
	op := constValue(t, "../../internal/operator/manifest.go", "PinnedOperatorVersion", "")

	assert.Equal(t, map[string]string{
		"library":      strings.TrimPrefix(lib, "v"),
		"core":         strings.TrimPrefix(core, "v"),
		"opm-operator": strings.TrimPrefix(op, "v"),
	}, got)
}

// TestPins_TestBinaryLinksTheLibrary confirms the test binary's build info
// reports the library dependency as go run does, so the test above checks
// the same reading the program makes.
func TestPins_TestBinaryLinksTheLibrary(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	require.True(t, ok)
	v, err := linkedVersion(info, libraryModule)
	require.NoError(t, err)
	assert.Equal(t, goModLibrary(t, "../../go.mod"), v)
}

// TestPins_GoRunPrintsTheSame runs the program as docs-kit.cue does and
// compares what it prints with pins(), so the build info go run embeds is
// the one the tests above read.
func TestPins_GoRunPrintsTheSame(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles the program")
	}
	out, err := exec.CommandContext(t.Context(), "go", "run", ".", "pins").Output()
	require.NoError(t, err)
	var doc struct {
		Schema string            `json:"schema"`
		Pins   map[string]string `json:"pins"`
	}
	require.NoError(t, json.Unmarshal(out, &doc))
	assert.Equal(t, "docs.opmodel.dev/pins/v1", doc.Schema)
	want, err := pins()
	require.NoError(t, err)
	assert.Equal(t, want, doc.Pins)
}

func TestLinkedVersion(t *testing.T) {
	tests := []struct {
		name    string
		dep     *debug.Module
		want    string
		wantErr string
	}{
		{name: "required", dep: &debug.Module{Path: libraryModule, Version: "v1.0.0-beta.2"}, want: "v1.0.0-beta.2"},
		{name: "replaced by a version", dep: &debug.Module{Path: libraryModule, Version: "v1.0.0-beta.1", Replace: &debug.Module{Path: "example.com/fork", Version: "v1.0.0-beta.3"}}, want: "v1.0.0-beta.3"},
		{name: "replaced by a directory", dep: &debug.Module{Path: libraryModule, Version: "v1.0.0-beta.1", Replace: &debug.Module{Path: "../library"}}, want: "(devel)"},
		{name: "absent", dep: &debug.Module{Path: "example.com/other", Version: "v1.0.0"}, wantErr: "is not in the build"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := linkedVersion(&debug.BuildInfo{Deps: []*debug.Module{tt.dep}}, libraryModule)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCoreRelease(t *testing.T) {
	v, err := coreRelease("opmodel.dev/core@v2.0.0-beta.2")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0-beta.2", v)

	for _, module := range []string{"opmodel.dev/core@v2", "opmodel.dev/core", "example.com/core@v2.0.0"} {
		_, err := coreRelease(module)
		assert.ErrorContains(t, err, module, "a module that pins no exact core release is refused, naming it")
	}
}

func TestRun_RefusesAnUnknownArgument(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"pin"}, &out)
	require.EqualError(t, err, "usage: docskit-dump [pins]")
	assert.Empty(t, out.String(), "nothing is printed on stdout")
}

// TestRun_DumpIsDeterministic catches what docs-kit's check (C14) refuses:
// two runs that print different bytes.
func TestRun_DumpIsDeterministic(t *testing.T) {
	for _, args := range [][]string{nil, {"pins"}} {
		var a, b bytes.Buffer
		require.NoError(t, run(args, &a))
		require.NoError(t, run(args, &b))
		assert.Equal(t, a.String(), b.String(), "args %v", args)

		var doc struct {
			Schema string `json:"schema"`
		}
		require.NoError(t, json.Unmarshal(a.Bytes(), &doc))
		assert.Contains(t, []string{"docs.opmodel.dev/cobradump/v1", "docs.opmodel.dev/pins/v1"}, doc.Schema)
	}
}

// goModLibrary returns the library version path's go.mod requires; a replace
// of the library is a failure.
func goModLibrary(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	f, err := modfile.Parse(path, data, nil)
	require.NoError(t, err)
	for _, r := range f.Replace {
		require.NotEqual(t, libraryModule, r.Old.Path, "go.mod replaces %s", libraryModule)
	}
	for _, r := range f.Require {
		if r.Mod.Path == libraryModule {
			return r.Mod.Version
		}
	}
	t.Fatalf("no require line for %s in %s", libraryModule, path)
	return ""
}

// moduleDir returns the module cache directory of the selected version of
// path.
func moduleDir(t *testing.T, path string) string {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), "go", "list", "-m", "-f", "{{.Dir}}", path).Output()
	require.NoError(t, err)
	dir := strings.TrimSpace(string(out))
	require.NotEmpty(t, dir, "go list found no directory for %s; run go mod download", path)
	return dir
}

// constValue returns the version in `name = "<prefix>v..."` in file, matched
// by its text, with the prefix removed.
func constValue(t *testing.T, file, name, prefix string) string {
	t.Helper()
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	re := regexp.MustCompile(`(?m)(?:^|[^A-Za-z0-9_])` + name + `\s*=\s*"` + prefix + `([^"]*)"`)
	m := re.FindAllStringSubmatch(string(data), -1)
	require.Len(t, m, 1, "exactly one %s in %s", name, file)
	return m[0][1]
}

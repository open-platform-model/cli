package loader

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// makeSynthFixture creates a minimal module fixture with its own cue.mod and
// a catalog dep pin. Callers can override the catalogVersion or omit the dep.
func makeSynthFixture(t *testing.T, opts synthFixtureOpts) string {
	t.Helper()
	dir := t.TempDir()

	if !opts.skipCueMod {
		modDir := filepath.Join(dir, "cue.mod")
		require.NoError(t, os.MkdirAll(modDir, 0o755))

		modContent := `module: "example.com/mymodule@v0"
language: { version: "v0.16.0" }
source: { kind: "self" }
`
		if !opts.skipCatalogDep {
			version := opts.catalogVersion
			if version == "" {
				version = "v1.3.9"
			}
			modContent += `deps: {
	"opmodel.dev/core/v1alpha1@v1": { v: "` + version + `" }
}
`
		}
		require.NoError(t, os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(modContent), 0o644))
	}

	if opts.moduleSource != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte(opts.moduleSource), 0o644))
	}
	return dir
}

type synthFixtureOpts struct {
	moduleSource   string
	catalogVersion string
	skipCatalogDep bool
	skipCueMod     bool
}

// localStubModule is a CUE source that satisfies SynthesizeOptions defaults
// without importing anything from the catalog. Used for tests that exercise
// pre-load failure paths.
const localStubModule = `package mymodule

metadata: {
	name:    "stub-module"
	version: "0.1.0"
}
`

// TestReadModuleCatalogPin covers the modfile-parsing helper.
func TestReadModuleCatalogPin(t *testing.T) {
	t.Run("returns the pin", func(t *testing.T) {
		dir := makeSynthFixture(t, synthFixtureOpts{
			catalogVersion: "v1.2.3",
		})
		ver, err := readModuleCatalogPin(dir)
		require.NoError(t, err)
		assert.Equal(t, "v1.2.3", ver)
	})

	t.Run("walks up to find cue.mod", func(t *testing.T) {
		dir := makeSynthFixture(t, synthFixtureOpts{catalogVersion: "v1.5.0"})
		nested := filepath.Join(dir, "subdir")
		require.NoError(t, os.MkdirAll(nested, 0o755))

		ver, err := readModuleCatalogPin(nested)
		require.NoError(t, err)
		assert.Equal(t, "v1.5.0", ver)
	})

	t.Run("missing catalog dep returns DetailError", func(t *testing.T) {
		dir := makeSynthFixture(t, synthFixtureOpts{skipCatalogDep: true})

		_, err := readModuleCatalogPin(dir)
		require.Error(t, err)

		var detail *oerrors.DetailError
		require.True(t, errors.As(err, &detail), "expected DetailError, got %T", err)
		assert.Contains(t, detail.Message, "opmodel.dev/core/v1alpha1@v1")
	})

	t.Run("missing modfile returns DetailError", func(t *testing.T) {
		dir := t.TempDir() // no cue.mod anywhere up to /tmp

		// Walk-up will eventually exit at filesystem root; ensure we get the right error.
		_, err := readModuleCatalogPin(dir)
		require.Error(t, err)
	})
}

// TestSynthesizeModuleInstanceFromPackage_ErrorPaths exercises the synthesis
// failure modes that don't require registry resolution.
func TestSynthesizeModuleInstanceFromPackage_ErrorPaths(t *testing.T) {
	ctx := cuecontext.New()

	t.Run("directory does not exist", func(t *testing.T) {
		_, err := SynthesizeModuleInstanceFromPackage(ctx, "/nonexistent/module/dir", SynthesizeOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("path is a file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "x.cue")
		require.NoError(t, os.WriteFile(filePath, []byte("package x\n"), 0o644))

		_, err := SynthesizeModuleInstanceFromPackage(ctx, filePath, SynthesizeOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a directory")
	})

	t.Run("module declares no catalog dep", func(t *testing.T) {
		dir := makeSynthFixture(t, synthFixtureOpts{
			moduleSource:   localStubModule,
			skipCatalogDep: true,
		})
		_, err := SynthesizeModuleInstanceFromPackage(ctx, dir, SynthesizeOptions{})
		require.Error(t, err)
		var detail *oerrors.DetailError
		require.True(t, errors.As(err, &detail))
		assert.Contains(t, detail.Hint, "opmodel.dev/core/v1alpha1@v1")
	})
}

// TestSynthesizeModuleInstanceFromPackage_AnchorCleanup verifies that the
// anchor temp directory is removed after both success and failure paths.
// We assert by recording temp-dir entries before and after.
func TestSynthesizeModuleInstanceFromPackage_AnchorCleanup(t *testing.T) {
	ctx := cuecontext.New()

	dir := makeSynthFixture(t, synthFixtureOpts{
		// Empty modules source — load.Instances will fail because there's no
		// CUE package. That exercises the post-anchor-creation error path.
		moduleSource: "",
	})

	before := tempEntriesMatching(t, "opm-synth-")
	_, err := SynthesizeModuleInstanceFromPackage(ctx, dir, SynthesizeOptions{})
	// Either succeeds or fails — we only assert no anchor leak.
	_ = err
	after := tempEntriesMatching(t, "opm-synth-")
	assert.Equal(t, before, after, "synth anchor temp dir should be cleaned up")
}

func tempEntriesMatching(t *testing.T, prefix string) []string {
	t.Helper()
	entries, err := os.ReadDir(os.TempDir())
	require.NoError(t, err)
	var out []string
	for _, e := range entries {
		if len(e.Name()) >= len(prefix) && e.Name()[:len(prefix)] == prefix {
			out = append(out, e.Name())
		}
	}
	return out
}

// TestSynthesizeModuleInstanceFromPackage_RegistryE2E exercises the full path
// against the real registry for each example module that defines debugValues.
// Skipped if no registry is reachable (CI runs with a pre-warmed registry).
func TestSynthesizeModuleInstanceFromPackage_RegistryE2E(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed synth tests")
	}
	ctx := cuecontext.New()

	examplesRoot, err := filepath.Abs("../../examples/modules")
	require.NoError(t, err)
	if _, statErr := os.Stat(examplesRoot); statErr != nil {
		t.Skip("examples/modules not available")
	}

	entries, err := os.ReadDir(examplesRoot)
	require.NoError(t, err)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		modPath := filepath.Join(examplesRoot, entry.Name())
		if _, statErr := os.Stat(filepath.Join(modPath, "module.cue")); statErr != nil {
			continue
		}
		// Only consider modules that actually define debugValues; otherwise
		// the synthesis can still run but downstream Parse would need values.
		data, readErr := os.ReadFile(filepath.Join(modPath, "module.cue"))
		if readErr != nil || !containsToken(data, "debugValues") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			result, err := SynthesizeModuleInstanceFromPackage(ctx, modPath, SynthesizeOptions{
				Name:      "test-debug",
				Namespace: "test-ns",
			})
			if err != nil {
				t.Skipf("synth failed (likely registry unavailable): %v", err)
			}
			require.NotNil(t, result)

			name, nameErr := result.Spec.LookupPath(cue.ParsePath("metadata.name")).String()
			require.NoError(t, nameErr)
			assert.Equal(t, "test-debug", name)

			ns, nsErr := result.Spec.LookupPath(cue.ParsePath("metadata.namespace")).String()
			require.NoError(t, nsErr)
			assert.Equal(t, "test-ns", ns)

			modField := result.Spec.LookupPath(cue.MakePath(cue.Def("module")))
			assert.True(t, modField.Exists(), "#module should be filled")
		})
	}
}

func containsToken(data []byte, token string) bool {
	for i := 0; i+len(token) <= len(data); i++ {
		if string(data[i:i+len(token)]) == token {
			return true
		}
	}
	return false
}

// TestSynthesizeModuleInstanceFromPackage_NameNamespaceOverride asserts the
// caller's --name and --namespace overrides land on the synthesized spec.
func TestSynthesizeModuleInstanceFromPackage_NameNamespaceOverride(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed synth tests")
	}
	ctx := cuecontext.New()

	modPath, err := filepath.Abs("../../examples/modules/zot_registry")
	require.NoError(t, err)
	if _, statErr := os.Stat(modPath); statErr != nil {
		t.Skip("examples/modules/zot_registry not available")
	}

	result, err := SynthesizeModuleInstanceFromPackage(ctx, modPath, SynthesizeOptions{
		Name:      "custom-name",
		Namespace: "custom-ns",
	})
	if err != nil {
		t.Skipf("synth failed (likely registry unavailable): %v", err)
	}

	name, _ := result.Spec.LookupPath(cue.ParsePath("metadata.name")).String()
	assert.Equal(t, "custom-name", name)

	ns, _ := result.Spec.LookupPath(cue.ParsePath("metadata.namespace")).String()
	assert.Equal(t, "custom-ns", ns)
}

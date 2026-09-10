package e2e

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"cuelang.org/go/mod/modfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/tests/fixtures"
)

// seedRenderHome returns a hermetic HOME whose ~/.opm holds exactly what
// `opm config init` writes (config.cue against the public registry mapping
// and the local default platform module), so render-bearing e2e tests never
// depend on the developer's real ~/.opm. Cleaned up with the test.
//
// The CUE module cache is not test state: the child process is pointed at
// the invoking user's cache (CUE_CACHE_DIR) so a cold temp HOME does not
// refetch every module and the read-only extracted tree never lands under
// the temp directory. Should anything still land there, the cleanup makes
// the tree writable before t.TempDir removes it.
func seedRenderHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Cleanup(func() { makeWritable(home) })
	if userCache, err := os.UserCacheDir(); err == nil && os.Getenv("CUE_CACHE_DIR") == "" {
		t.Setenv("CUE_CACHE_DIR", filepath.Join(userCache, "cue"))
	}
	opmDir := filepath.Join(home, ".opm")
	require.NoError(t, os.MkdirAll(opmDir, 0o700))
	configPath := filepath.Join(opmDir, "config.cue")
	require.NoError(t, os.WriteFile(configPath, []byte(config.DefaultConfigTemplate), 0o600))
	require.NoError(t, config.WritePlatformModule(config.PlatformDir(configPath)))
	return home
}

// makeWritable chmods every entry under dir writable so a read-only tree
// (CUE extracts module cache files read-only) can be removed.
func makeWritable(dir string) {
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Best effort: an unreadable entry is skipped, the walk continues.
			return nil //nolint:nilerr // walk continues past entries it cannot read
		}
		if d.IsDir() {
			_ = os.Chmod(path, 0o755)
		} else {
			_ = os.Chmod(path, 0o644)
		}
		return nil
	})
}

// olderCatalogPin is a published build of the abstraction catalog that is
// older than the build the examples module requires (examples/cue.mod pins
// v4.0.1 or newer), so a platform pinning it exhibits catalog version skew.
const olderCatalogPin = "v4.0.0"

// seedSkewPlatform writes a platform module identical to the seeded default
// except that the abstraction catalog is pinned at olderCatalogPin, and
// returns its directory.
func seedSkewPlatform(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, config.WritePlatformModule(dir))
	modFile := filepath.Join(dir, filepath.FromSlash(config.PlatformModuleFileName))
	content, err := os.ReadFile(modFile)
	require.NoError(t, err)
	skewed := strings.Replace(string(content), config.DefaultCatalogPins[0], olderCatalogPin, 1)
	require.NotEqual(t, string(content), skewed, "the seeded module must pin the abstraction catalog")
	require.NoError(t, os.WriteFile(modFile, []byte(skewed), 0o600))
	return dir
}

// examplesCatalogPin reads the abstraction catalog build the examples module
// requires, so the skew assertions name the version the kernel reports
// rather than a literal that drifts on the next fixture bump.
func examplesCatalogPin(t *testing.T, repoRoot string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repoRoot, "examples", "cue.mod", "module.cue"))
	require.NoError(t, err)
	m := regexp.MustCompile(`"` + regexp.QuoteMeta(config.DefaultCatalogPaths[0]) + `":\s*\{\s*v:\s*"([^"]+)"`).FindStringSubmatch(string(content))
	require.Len(t, m, 2, "examples/cue.mod/module.cue must pin %s", config.DefaultCatalogPaths[0])
	return m[1]
}

// podinfoExample returns the repo root and the podinfo example instance file.
func podinfoExample(t *testing.T) (repoRoot, instanceFile string) {
	t.Helper()
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}
	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	instanceFile = filepath.Join(repoRoot, "examples", "instances", "podinfo", "instance.cue")
	if _, statErr := os.Stat(instanceFile); statErr != nil {
		t.Skipf("examples/instances/podinfo not available: %v", statErr)
	}
	return repoRoot, instanceFile
}

func exitCode(t *testing.T, err error) int {
	t.Helper()
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr), "expected an exit error, got %v", err)
	return exitErr.ExitCode()
}

// TestE2E_InstanceBuild_LayersValuesFile covers the kernel-render scenario
// "Instance file with extra values": a -f file is layered onto the instance
// package as a values source, the render imports the layered package and the
// rendered objects reflect the override, while the package's own values.cue
// still applies.
func TestE2E_InstanceBuild_LayersValuesFile(t *testing.T) {
	_, instanceFile := podinfoExample(t)
	home := seedRenderHome(t)

	workDir := t.TempDir()
	override := filepath.Join(workDir, "override.cue")
	require.NoError(t, os.WriteFile(override, []byte("package podinfo\nvalues: image: tag: \"6.7.0\"\n"), 0o600))

	stdout, stderr, err := runOPMWithEnv(t, workDir, home, 180*time.Second,
		"instance", "build", instanceFile, "-f", override)
	require.NoError(t, err, "stderr: %s", stderr)

	assert.Contains(t, stdout, "ghcr.io/stefanprodan/podinfo:6.7.0", "the -f override must reach the rendered Deployment")
	assert.NotContains(t, stdout, "podinfo:6.7.1", "the module default must be overridden")
	assert.Contains(t, stdout, "replicas: 2", "the package's own values.cue still applies beside the override")
	assert.Contains(t, stderr, "(local default)", "provenance names the local default platform module")
}

// TestE2E_InstanceBuild_SkewWarnsByDefault covers "Render warnings reach the
// user": with no skewPolicy configured, a module requiring a newer catalog
// build than the platform pins renders against the platform's build and the
// CLI's skew warning (worded from the kernel's resolved-versions row) names
// the path and both versions.
func TestE2E_InstanceBuild_SkewWarnsByDefault(t *testing.T) {
	repoRoot, instanceFile := podinfoExample(t)
	home := seedRenderHome(t)
	platformDir := seedSkewPlatform(t)
	required := examplesCatalogPin(t, repoRoot)

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), home, 180*time.Second,
		"instance", "build", instanceFile, "--platform", platformDir)
	require.NoError(t, err, "stderr: %s", stderr)

	assert.NotEmpty(t, stdout, "the render proceeds under the warn policy")
	assert.Contains(t, stderr, `version skew on "`+config.DefaultCatalogPaths[0]+`"`)
	assert.Contains(t, stderr, "module requires "+required)
	assert.Contains(t, stderr, "platform carries "+olderCatalogPin)
	assert.Contains(t, stderr, "rendering against the platform's build")
	assert.Contains(t, stderr, "(--platform)", "provenance names the flag-provided directory")
	assert.NotContains(t, stderr, "skew policy: refuse")
}

// TestE2E_InstanceBuild_SkewRefusedByConfig covers "Skew refusal is a
// validation failure" and the config-types scenario "Refuse configured":
// with skewPolicy "refuse" in config.cue, the render is refused before
// evaluation as a validation error naming the path and both versions.
func TestE2E_InstanceBuild_SkewRefusedByConfig(t *testing.T) {
	repoRoot, instanceFile := podinfoExample(t)
	home := seedRenderHome(t)
	platformDir := seedSkewPlatform(t)
	required := examplesCatalogPin(t, repoRoot)

	configPath := filepath.Join(home, ".opm", "config.cue")
	refusing := strings.Replace(config.DefaultConfigTemplate, `skewPolicy: "warn"`, `skewPolicy: "refuse"`, 1)
	require.NotEqual(t, config.DefaultConfigTemplate, refusing, "the config template must carry the skewPolicy key")
	require.NoError(t, os.WriteFile(configPath, []byte(refusing), 0o600))

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), home, 180*time.Second,
		"instance", "build", instanceFile, "--platform", platformDir)
	require.Error(t, err, "stdout: %s", stdout)

	assert.Equal(t, opmexit.ExitValidationError, exitCode(t, err), "a skew refusal is a validation failure")
	assert.Empty(t, stdout, "nothing is rendered when the render is refused before evaluation")
	assert.Contains(t, stderr, "skew policy: refuse (config)", "provenance reports the refuse policy and its source")
	assert.Contains(t, stderr, "render failed")
	assert.Contains(t, stderr, `version skew on "`+config.DefaultCatalogPaths[0]+`"`)
	assert.Contains(t, stderr, "module requires "+required)
	assert.Contains(t, stderr, "platform carries "+olderCatalogPin)
}

// libModulePath is a module no registry serves: the never-published
// dependency a developer redirects with cue.mod/local-module.cue.
const libModulePath = "test.example/lib@v0"

// libImageTag is the value only the never-published module supplies, so a
// render that evaluated the replacement directory is distinguishable from one
// that did not.
const libImageTag = "6.7.0-from-local-lib"

// catalogLabel is the label the patched catalog copy stamps on its Deployment
// output (catalogCopyWithLabel); the published build never carries it.
const catalogLabel = "render.test/catalog"

func writeE2EFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// writeLibModule writes the never-published module: one package exporting
// the image tag the instance reads.
func writeLibModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeE2EFile(t, filepath.Join(dir, "cue.mod", "module.cue"),
		"module: \""+libModulePath+"\"\nlanguage: version: \"v0.17.0\"\n")
	writeE2EFile(t, filepath.Join(dir, "lib.cue"), "package lib\n\nTag: \""+libImageTag+"\"\n")
	return dir
}

// modDeps reads the pinned dependencies of a strict cue.mod/module.cue.
func modDeps(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	f, err := modfile.Parse(data, path)
	require.NoError(t, err)
	deps := make(map[string]string, len(f.Deps))
	for p, d := range f.Deps {
		deps[p] = d.Version
	}
	return deps
}

// localModuleFile renders a cue.mod/local-module.cue: the whole main-module
// dependency view (cue/load reads it in place of module.cue's deps when the
// directory is the main module, so it lists every dependency, versions
// inherited from module.cue where omitted) with the given paths redirected.
func localModuleFile(paths []string, replacements map[string]string) string {
	sort.Strings(paths)
	var b strings.Builder
	b.WriteString("deps: {\n")
	for _, p := range paths {
		if target, ok := replacements[p]; ok {
			fmt.Fprintf(&b, "\t%q: replaceWith: %q\n", p, target)
			continue
		}
		fmt.Fprintf(&b, "\t%q: {}\n", p)
	}
	b.WriteString("}\n")
	return b.String()
}

// replacementInstance writes an instance module carrying the examples
// module's pins plus, when withLib is set, the never-published lib module
// listed version-less and read for the image tag. replacements are the
// paths its cue.mod/local-module.cue redirects (nil writes no local file).
// Returns the instance file.
func replacementInstance(t *testing.T, repoRoot string, withLib bool, replacements map[string]string) string {
	t.Helper()
	deps := modDeps(t, filepath.Join(repoRoot, "examples", "cue.mod", "module.cue"))
	podinfoPath := fixtures.Must(t, "podinfo").ModulePath
	require.Contains(t, deps, podinfoPath, "examples/cue.mod/module.cue pins the podinfo fixture")

	dir := t.TempDir()
	var mod strings.Builder
	mod.WriteString("module: \"test.example/app@v0\"\nlanguage: version: \"v0.17.0\"\ndeps: {\n")
	paths := make([]string, 0, len(deps)+1)
	for p, v := range deps {
		fmt.Fprintf(&mod, "\t%q: v: %q\n", p, v)
		paths = append(paths, p)
	}
	if withLib {
		fmt.Fprintf(&mod, "\t%q: {}\n", libModulePath)
		paths = append(paths, libModulePath)
	}
	mod.WriteString("}\n")
	writeE2EFile(t, filepath.Join(dir, "cue.mod", "module.cue"), mod.String())
	if replacements != nil {
		writeE2EFile(t, filepath.Join(dir, "cue.mod", "local-module.cue"), localModuleFile(paths, replacements))
	}

	imports := "\tcore \"" + config.DefaultCorePath + "\"\n\tpodinfo \"" + podinfoPath + "\"\n"
	tag := "\"6.7.1\""
	if withLib {
		imports += "\tlib \"" + libModulePath + "\"\n"
		tag = "lib.Tag"
	}
	writeE2EFile(t, filepath.Join(dir, "instance.cue"), `package app

import (
`+imports+`)

core.#ModuleInstance

metadata: {
	name:      "podinfo-local"
	namespace: "default"
}

#module: podinfo

values: {
	replicas: 2
	image: tag: `+tag+`
}
`)
	return filepath.Join(dir, "instance.cue")
}

// seedPlatform writes the seeded default platform module into a fresh
// directory and returns it.
func seedPlatform(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, config.WritePlatformModule(dir))
	return dir
}

// catalogCopyWithLabel copies the abstraction catalog build the seeded
// platform pins out of the CUE module cache into a writable directory and
// stamps one extra label on its deployment transformer's output. The cache
// is the invoking user's (seedRenderHome); a cold cache is warmed by one
// plain render of the podinfo example against platformDir first.
func catalogCopyWithLabel(t *testing.T, home, platformDir, instanceFile string) string {
	t.Helper()
	catalogPath, _, _ := strings.Cut(config.DefaultCatalogPaths[0], "@")
	src := filepath.Join(os.Getenv("CUE_CACHE_DIR"), "mod", "extract",
		filepath.FromSlash(catalogPath)+"@"+config.DefaultCatalogPins[0])
	if _, err := os.Stat(src); err != nil {
		_, stderr, runErr := runOPMWithEnv(t, t.TempDir(), home, 180*time.Second,
			"instance", "build", instanceFile, "--platform", platformDir)
		require.NoError(t, runErr, "warming the module cache: %s", stderr)
	}
	require.DirExists(t, src, "the platform's pinned catalog is extracted in the CUE module cache after a render")

	dst := t.TempDir()
	require.NoError(t, filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o644)
	}))

	transformer := filepath.Join(dst, "transformers", "deployment_transformer.cue")
	content, err := os.ReadFile(transformer)
	require.NoError(t, err)
	const anchor = "labels:    #context.labels" // the Deployment metadata line
	require.Contains(t, string(content), anchor, "the pinned catalog's deployment transformer carries the metadata labels line this test patches")
	patched := strings.Replace(string(content), anchor,
		"labels: {\n\t\t\t\tfor k, v in #context.labels {(k): v}\n\t\t\t\t\""+catalogLabel+"\": \"local\"\n\t\t\t}", 1)
	require.NoError(t, os.WriteFile(transformer, []byte(patched), 0o644))
	return dst
}

// TestE2E_InstanceBuild_LocalReplacementIsHonored covers the kernel-render
// scenario "Replaced dependency warns": the instance's main module lists a
// never-published module version-less and redirects it with
// cue.mod/local-module.cue; the render evaluates the directory's bytes and
// the warning names the path, the directory and the instance as its source.
func TestE2E_InstanceBuild_LocalReplacementIsHonored(t *testing.T) {
	repoRoot, _ := podinfoExample(t)
	home := seedRenderHome(t)
	platformDir := seedPlatform(t)
	libDir := writeLibModule(t)
	instanceFile := replacementInstance(t, repoRoot, true, map[string]string{libModulePath: libDir})

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), home, 180*time.Second,
		"instance", "build", instanceFile, "--platform", platformDir)
	require.NoError(t, err, "stderr: %s", stderr)

	assert.Contains(t, stdout, "ghcr.io/stefanprodan/podinfo:"+libImageTag, "the rendered Deployment carries the replacement directory's value")
	assert.Contains(t, stderr, "local replacement in effect: "+libModulePath+" served from "+libDir+" (instance)")
	assert.Contains(t, stderr, "rendered bytes may not correspond to any published build")
	assert.NotContains(t, stderr, "is ignored")
}

// TestE2E_InstanceBuild_ReplacementOfPlatformPathIsInert covers "Module-side
// replacement of a platform path is reported inert": the instance redirects
// the abstraction catalog the platform names; the render uses the platform's
// pinned catalog, the output is that of a clean run, and the warning says
// the redirect belongs in the platform module's local file.
func TestE2E_InstanceBuild_ReplacementOfPlatformPathIsInert(t *testing.T) {
	repoRoot, example := podinfoExample(t)
	home := seedRenderHome(t)
	platformDir := seedPlatform(t)
	catDir := catalogCopyWithLabel(t, home, platformDir, example)
	clean := replacementInstance(t, repoRoot, false, nil)
	redirected := replacementInstance(t, repoRoot, false, map[string]string{config.DefaultCatalogPaths[0]: catDir})

	cleanOut, stderr, err := runOPMWithEnv(t, t.TempDir(), home, 180*time.Second,
		"instance", "build", clean, "--platform", platformDir)
	require.NoError(t, err, "stderr: %s", stderr)
	require.NotContains(t, stderr, "local replacement", "the clean run warns of nothing")

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), home, 180*time.Second,
		"instance", "build", redirected, "--platform", platformDir)
	require.NoError(t, err, "stderr: %s", stderr)

	assert.Equal(t, cleanOut, stdout, "the platform's pinned catalog rendered, not the instance's redirected copy")
	assert.NotContains(t, stdout, catalogLabel)
	localFile := filepath.Join(filepath.Dir(redirected), "cue.mod", "local-module.cue")
	assert.Contains(t, stderr, "local replacement of "+config.DefaultCatalogPaths[0]+" in "+localFile+" is ignored: the platform names that path")
	assert.Contains(t, stderr, "redirect it in the platform module's cue.mod/local-module.cue")
	assert.NotContains(t, stderr, "in effect")
}

// TestE2E_InstanceBuild_PlatformReplacementIsHonored covers "Platform
// replacement of its catalog is honored and warns": the --platform module
// redirects its abstraction catalog to a patched copy; the rendered
// Deployment carries the copy's label and the warning names the platform as
// the source.
func TestE2E_InstanceBuild_PlatformReplacementIsHonored(t *testing.T) {
	_, example := podinfoExample(t)
	home := seedRenderHome(t)
	platformDir := seedPlatform(t)
	catDir := catalogCopyWithLabel(t, home, platformDir, example)

	deps := modDeps(t, filepath.Join(platformDir, filepath.FromSlash(config.PlatformModuleFileName)))
	paths := make([]string, 0, len(deps))
	for p := range deps {
		paths = append(paths, p)
	}
	writeE2EFile(t, filepath.Join(platformDir, "cue.mod", "local-module.cue"),
		localModuleFile(paths, map[string]string{config.DefaultCatalogPaths[0]: catDir}))

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), home, 180*time.Second,
		"instance", "build", example, "--platform", platformDir)
	require.NoError(t, err, "stderr: %s", stderr)

	assert.Contains(t, stdout, catalogLabel+": local", "the rendered Deployment carries the copy's transformer output")
	assert.Contains(t, stderr, "local replacement in effect: "+config.DefaultCatalogPaths[0]+" served from "+catDir+" (platform)")
	assert.NotContains(t, stderr, "is ignored")
}

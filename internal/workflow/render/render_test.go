package render

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	libmodule "github.com/open-platform-model/library/opm/module"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/platform"
	"github.com/open-platform-model/cli/pkg/module"
)

func mustInstanceMetadata(name, namespace string) module.InstanceMetadata {
	return module.InstanceMetadata{Name: name, Namespace: namespace}
}

func TestShowRenderOutput_NoErrors_DefaultMode(t *testing.T) {
	result := &Result{Instance: mustInstanceMetadata("demo", "default")}
	assert.NotPanics(t, func() { ShowOutput(result, ShowOutputOpts{}) })
}

func TestShowRenderOutput_Warnings(t *testing.T) {
	result := &Result{Instance: mustInstanceMetadata("demo", "default"), Warnings: []string{"w1"}}
	assert.NotPanics(t, func() { ShowOutput(result, ShowOutputOpts{Verbose: true}) })
}

func TestRenderResult_HasWarnings(t *testing.T) {
	assert.False(t, (&Result{}).HasWarnings())
	assert.True(t, (&Result{Warnings: []string{"x"}}).HasWarnings())
}

func TestRenderResult_ResourceCount(t *testing.T) {
	assert.Equal(t, 0, (&Result{}).ResourceCount())
}

func TestRenderFromInstanceFile_NilConfig(t *testing.T) {
	_, err := FromInstanceFile(context.Background(), InstanceFileOpts{InstanceFilePath: "instance.cue", Config: nil, K8sConfig: nil})
	require.Error(t, err)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "configuration not loaded")
}

func TestRenderFromInstanceFile_NilK8sConfig(t *testing.T) {
	_, err := FromInstanceFile(context.Background(), InstanceFileOpts{InstanceFilePath: "instance.cue", Config: &config.GlobalConfig{}, K8sConfig: nil})
	require.Error(t, err)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "kubernetes config not resolved")
}

func TestRenderFromInstanceFile_RejectsModulePackagePath(t *testing.T) {
	// The path guard fires before platform resolution, so no registry or
	// platform module is needed.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte("package test\n"), 0o644))

	_, err := FromInstanceFile(context.Background(), InstanceFileOpts{
		InstanceFilePath: dir,
		Config:           &config.GlobalConfig{},
		K8sConfig:        &config.ResolvedKubernetesConfig{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module package, not an instance")
}

func TestNewResult_CarriesResolvedPlatform(t *testing.T) {
	// The apply workflow seeds the cluster Platform from Result.PlatformSpec
	// (0006 D12): the assembly must carry the spec decoded from the built
	// platform verbatim — every entry with its derived version — or the
	// seeded document degrades to the zero value.
	spec := platform.Spec{
		Name: "cluster",
		Type: "kubernetes",
		Entries: []platform.Entry{
			{Path: "opmodel.dev/catalogs/k8s@v1", Version: "1.0.0-alpha.2", Enable: true},
			{Path: "opmodel.dev/catalogs/opm@v4", Version: "4.0.1", Enable: true},
		},
	}
	env := &renderEnv{
		resolution: platform.Resolution{Source: platform.SourceLocalDefault, Location: "/home/x/.opm/platform", Dir: "/home/x/.opm/platform", Warning: "cluster Platform not found"},
		spec:       spec,
	}
	out := &kernel.RenderResult{
		Diagnostics: kernel.RenderDiagnostics{
			Pairs:           []kernel.RenderPair{{Component: "web", Transformer: "x#Deployment"}},
			UnhandledTraits: map[string][]string{"web": {"t@v1"}},
		},
	}

	result := newResult(env, out, "digest", map[string]any{"k": "v"}, true)

	assert.Equal(t, env.resolution, result.Platform)
	assert.Equal(t, spec, result.PlatformSpec)
	assert.NotEmpty(t, result.PlatformSpec.Type, "seeded spec must never carry an empty type")
	for _, e := range result.PlatformSpec.Entries {
		assert.NotEmpty(t, e.Version, "seed carries the derived version of %s", e.Path)
	}
	assert.Equal(t, formatAdvisories(out.Diagnostics), result.Warnings, "warnings are the CLI's wording of the advisory rows")
	assert.Len(t, result.Warnings, 1)
	assert.Equal(t, out.Diagnostics.Pairs, result.Pairs)
	assert.Equal(t, "digest", result.RenderDigest)
	assert.Equal(t, map[string]any{"k": "v"}, result.Values)
	assert.True(t, result.SourceLocal)
}

func TestSkewPolicyFor(t *testing.T) {
	cr := func(policy string) platform.Resolution {
		return platform.Resolution{Source: platform.SourceClusterCR, SkewPolicy: policy}
	}
	local := platform.Resolution{Source: platform.SourceLocalDefault}
	flag := platform.Resolution{Source: platform.SourceFlagDir}
	warnCfg := &config.GlobalConfig{SkewPolicy: config.SkewPolicyWarn}
	refuseCfg := &config.GlobalConfig{SkewPolicy: config.SkewPolicyRefuse}
	absentCfg := &config.GlobalConfig{}

	tests := []struct {
		name     string
		res      platform.Resolution
		cfg      *config.GlobalConfig
		want     kernel.SkewPolicy
		wantNote string
	}{
		{"local default is warn", local, absentCfg, kernel.SkewWarn, ""},
		{"local with warn key", local, warnCfg, kernel.SkewWarn, ""},
		{"local with refuse key", local, refuseCfg, kernel.SkewRefuse, "refuse (config)"},
		{"flag with refuse key", flag, refuseCfg, kernel.SkewRefuse, "refuse (config)"},
		{"cluster unset is warn", cr(""), absentCfg, kernel.SkewWarn, ""},
		{"cluster Warn", cr("Warn"), absentCfg, kernel.SkewWarn, ""},
		{"cluster Refuse", cr("Refuse"), warnCfg, kernel.SkewRefuse, "refuse (cluster Platform)"},
		{"cluster Warn overrides refuse key", cr("Warn"), refuseCfg, kernel.SkewWarn, "cluster Platform overrides config skewPolicy"},
		{"cluster unset overrides refuse key", cr(""), refuseCfg, kernel.SkewWarn, "cluster Platform overrides config skewPolicy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, note := skewPolicyFor(tt.res, tt.cfg)
			assert.Equal(t, tt.want, got)
			if tt.wantNote == "" {
				assert.Empty(t, note)
			} else {
				assert.Contains(t, note, tt.wantNote)
			}
		})
	}
}

func TestLoadValuesSources_Empty(t *testing.T) {
	sources, err := loadValuesSources(kernel.New(), nil)
	require.NoError(t, err)
	assert.Nil(t, sources, "no files means no sources: the package's own values apply")
}

func TestLoadValuesSources_InOrderWithFileOrigin(t *testing.T) {
	k := kernel.New()
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.cue")
	f2 := filepath.Join(dir, "b.cue")
	require.NoError(t, os.WriteFile(f1, []byte("package test\nvalues: {replicas: 3}\n"), 0o644))
	require.NoError(t, os.WriteFile(f2, []byte("package test\nvalues: {image: \"nginx\"}\n"), 0o644))

	sources, err := loadValuesSources(k, []string{f1, f2})
	require.NoError(t, err)
	require.Len(t, sources, 2)
	assert.Equal(t, f1, sources[0].Origin, "each source is attributed to its file")
	assert.Equal(t, f2, sources[1].Origin)
	// A Source carries bytes the kernel compiles where it uses them; the
	// top-level values field is unwrapped at that point.
	schema := cuecontext.New().CompileString(`{replicas: int, image: string}`)
	require.NoError(t, schema.Err())
	merged, err := k.ValidateConfigDetailed(schema, sources)
	require.NoError(t, err, "the top-level values field is unwrapped")
	replicas, err := merged.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas)
}

func TestLoadValuesSources_MissingFileNamesIt(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.cue")
	_, err := loadValuesSources(kernel.New(), []string{missing})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nope.cue")
}

func writeD19File(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// The CLI's one render call always opts into the kernel's local replacements
// (the D19 spec: every render enables them): a developer's local-module.cue
// is honored, never refused, and the kernel's rows are what the warnings
// are worded from.
func TestNewRenderInput_EnablesLocalReplacements(t *testing.T) {
	env := &renderEnv{skew: kernel.SkewRefuse}
	inst := &libmodule.Instance{}

	in := newRenderInput(env, inst)

	assert.True(t, in.LocalReplacements, "every CLI render opts into local replacements")
	assert.Same(t, inst, in.Instance)
	assert.Equal(t, RuntimeName, in.RuntimeName)
	assert.Equal(t, kernel.SkewRefuse, in.Skew, "the resolved skew policy is carried through")
}

func TestModuleContextRoot_WalksUpToTheModuleRoot(t *testing.T) {
	root := t.TempDir()
	writeD19File(t, filepath.Join(root, "cue.mod", "module.cue"), `module: "example.com/main@v0"`)
	nested := filepath.Join(root, "instances", "web")
	require.NoError(t, os.MkdirAll(nested, 0o755))

	assert.Equal(t, root, moduleContextRoot(nested), "an instance file's directory resolves to its module root")
	assert.Equal(t, root, moduleContextRoot(root), "a module directory is its own root")
}

func TestModuleContextRoot_NoModuleIsEmpty(t *testing.T) {
	assert.Equal(t, "", moduleContextRoot(t.TempDir()), "no cue.mod above the directory: no module context")
}

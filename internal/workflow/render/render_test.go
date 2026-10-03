package render

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/objectset"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"

	"github.com/open-platform-model/cli/internal/cmdutil/cmdutiltest"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/platform"
	pkgerrors "github.com/open-platform-model/cli/pkg/errors"
)

func TestShowRenderOutput_NoErrors_DefaultMode(t *testing.T) {
	result := &Result{Instance: module.InstanceMetadata{Name: "demo", Namespace: "default"}}
	assert.NotPanics(t, func() { ShowOutput(result, ShowOutputOpts{}) })
}

func TestShowRenderOutput_Warnings(t *testing.T) {
	result := &Result{Instance: module.InstanceMetadata{Name: "demo", Namespace: "default"}, Warnings: []string{"w1"}}
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

// A module package is refused by kind, before platform resolution, so no
// registry or platform module is needed; the refusal names the module
// command the caller passed. A module file argument is judged by its package.
func TestRenderFromInstanceFile_RefusesModulePackage(t *testing.T) {
	dir := cmdutiltest.WriteMinimalModule(t)
	for _, arg := range []string{dir, filepath.Join(dir, "module.cue")} {
		_, err := FromInstanceFile(context.Background(), InstanceFileOpts{
			InstanceFilePath: arg,
			ModuleCommand:    "opm module build",
			Config:           &config.GlobalConfig{},
			K8sConfig:        &config.ResolvedKubernetesConfig{},
		})
		var exitErr *opmexit.ExitError
		require.True(t, errors.As(err, &exitErr), arg)
		assert.Equal(t, opmexit.ExitValidationError, exitErr.Code, arg)
		assert.Contains(t, err.Error(), dir+" is a module, not an instance", arg)
		assert.Contains(t, err.Error(), "opm module build "+dir, arg)
	}
}

// An instance package under no CUE module is refused as a validation error
// naming the directory, before the acquire: the refusal is the design's, not
// the loader's unresolved-import error, with or without --platform.
func TestRenderFromInstanceFile_RefusesPackageUnderNoModuleRoot(t *testing.T) {
	dir := t.TempDir()
	require.Empty(t, moduleContextRoot(dir), "the temp dir must sit under no cue.mod")
	writeD19File(t, filepath.Join(dir, "instance.cue"),
		"package hello\n\nimport m \"example.com/modules/hello@v0\"\n\nm\n")
	for _, platformFlag := range []string{"", filepath.Join(t.TempDir(), "platform")} {
		_, err := FromInstanceFile(context.Background(), InstanceFileOpts{
			InstanceFilePath: dir,
			PlatformFlag:     platformFlag,
			Config:           &config.GlobalConfig{},
			K8sConfig:        &config.ResolvedKubernetesConfig{},
		})
		var exitErr *opmexit.ExitError
		require.True(t, errors.As(err, &exitErr), "platform %q", platformFlag)
		assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
		assert.Contains(t, err.Error(), "instance package "+dir+" is under no CUE module")
	}
}

func TestInstanceContext_DirectoryWithItsOwnCueMod(t *testing.T) {
	// An instance package directory inside a module tree that carries its
	// own cue.mod: the directory is its own context, not the tree's root.
	outer := t.TempDir()
	writeD19File(t, filepath.Join(outer, "cue.mod", "module.cue"), `module: "example.com/outer@v0"`)
	inst := filepath.Join(outer, "instances", "hello")
	writeD19File(t, filepath.Join(inst, "cue.mod", "module.cue"), `module: "example.com/hello@v0"`)

	dir, root, err := instanceContext(inst)
	require.NoError(t, err)
	assert.Equal(t, inst, dir)
	assert.Equal(t, inst, root)
}

func TestInstanceContext_FileArgument(t *testing.T) {
	root := t.TempDir()
	writeD19File(t, filepath.Join(root, "cue.mod", "module.cue"), `module: "example.com/hello@v0"`)
	file := filepath.Join(root, "instance.cue")
	writeD19File(t, file, "package hello\n")

	dir, gotRoot, err := instanceContext(file)
	require.NoError(t, err)
	assert.Equal(t, root, dir)
	assert.Equal(t, root, gotRoot)
}

func TestNewResult_CarriesResolvedPlatform(t *testing.T) {
	env := &renderEnv{
		resolution: platform.Resolution{Source: platform.SourceModuleDeps, DepsKind: platform.DepsInstance, Dir: "/home/x/.opm/cache/platforms/abc", Warning: "cluster Platform not used (no Platform CR in the cluster)"},
	}
	out := &kernel.RenderResult{
		Diagnostics: kernel.RenderDiagnostics{
			Pairs:           []kernel.RenderPair{{Component: "web", Transformer: "x#Deployment"}},
			UnhandledTraits: map[string][]string{"web": {"t@v1"}},
		},
	}

	result := newResult(env, out, "digest", map[string]any{"k": "v"}, true)

	assert.Equal(t, env.resolution, result.Platform)
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
	arg := platform.Resolution{Source: platform.SourceArgumentDir}
	flag := platform.Resolution{Source: platform.SourceFlagDir}
	deps := platform.Resolution{Source: platform.SourceModuleDeps}
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
		{"flag default is warn", flag, absentCfg, kernel.SkewWarn, ""},
		{"flag with warn key", flag, warnCfg, kernel.SkewWarn, ""},
		{"argument with refuse key", arg, refuseCfg, kernel.SkewRefuse, "refuse (config)"},
		{"flag with refuse key", flag, refuseCfg, kernel.SkewRefuse, "refuse (config)"},
		{"cluster unset is warn", cr(""), absentCfg, kernel.SkewWarn, ""},
		{"cluster Warn", cr("Warn"), absentCfg, kernel.SkewWarn, ""},
		{"cluster Refuse", cr("Refuse"), warnCfg, kernel.SkewRefuse, "refuse (cluster Platform)"},
		{"cluster Warn overrides refuse key", cr("Warn"), refuseCfg, kernel.SkewWarn, "cluster Platform overrides config skewPolicy"},
		{"cluster unset overrides refuse key", cr(""), refuseCfg, kernel.SkewWarn, "cluster Platform overrides config skewPolicy"},
		{"module deps is warn", deps, absentCfg, kernel.SkewWarn, ""},
		{"module deps ignores and never names the refuse key", deps, refuseCfg, kernel.SkewWarn, ""},
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
// (the 0010:D19 spec: every render enables them): a developer's local-module.cue
// is honored, never refused, and the kernel's rows are what the warnings
// are worded from.
func TestNewRenderInput_EnablesLocalReplacements(t *testing.T) {
	env := &renderEnv{skew: kernel.SkewRefuse}
	inst := &module.Instance{}

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

// compiledObject builds one rendered object the way the kernel hands it to
// the render workflow: a concrete CUE value plus its component provenance.
func compiledObject(t *testing.T, component, src string) *kernel.Compiled {
	t.Helper()
	v := cuecontext.New().CompileString(src)
	require.NoError(t, v.Err())
	return &kernel.Compiled{
		Value:       v,
		Instance:    "backup-system",
		Component:   component,
		Transformer: "opmodel.dev/catalogs/opm/transformers/transformer-registration-transformer@4.4.0",
	}
}

// registrationObject is the object a transformer-registration component
// renders: its name is derived from the instance, so two such components in
// one module render it twice.
const registrationObject = `{
	apiVersion: "opmodel.dev/v1alpha1"
	kind:       "TransformerRegistration"
	metadata: {name: "backup-system.k8up", namespace: "backup-system"}
}`

// A module shipping two transformer-registration components renders two
// TransformerRegistration objects under one instance-derived name; the last
// apply would silently overwrite the first, so the render is refused naming
// the identity and both producers.
func TestRefuseDuplicateIdentities_TwoRegistrations(t *testing.T) {
	out := &kernel.RenderResult{Compiled: []*kernel.Compiled{
		compiledObject(t, "registration", registrationObject),
		compiledObject(t, "registration-copy", registrationObject),
	}}

	err := refuseDuplicateIdentities(out)

	require.Error(t, err)
	var dupErr *objectset.DuplicateIdentitiesError
	require.ErrorAs(t, err, &dupErr)
	require.Len(t, dupErr.Duplicates, 1)
	assert.Equal(t, "TransformerRegistration", dupErr.Duplicates[0].Identity.Kind)
	assert.Equal(t, "backup-system.k8up", dupErr.Duplicates[0].Identity.Name)
	assert.Contains(t, err.Error(), "component \"registration\"")
	assert.Contains(t, err.Error(), "component \"registration-copy\"")
}

// Every render whose objects address distinct apply identities proceeds
// unchanged: the check is the only thing between a successful render and the
// resources built from it.
func TestRefuseDuplicateIdentities_DistinctIdentities(t *testing.T) {
	out := &kernel.RenderResult{Compiled: []*kernel.Compiled{
		compiledObject(t, "web", `{
	apiVersion: "apps/v1"
	kind:       "Deployment"
	metadata: {name: "web", namespace: "default"}
}`),
		compiledObject(t, "web", `{
	apiVersion: "v1"
	kind:       "Service"
	metadata: {name: "web", namespace: "default"}
}`),
	}}

	assert.NoError(t, refuseDuplicateIdentities(out))
}

// A value carrying no metadata.name is not a Kubernetes object: the helper
// skips it rather than refusing on it, and the real duplicate beside it is
// still the only row.
func TestRefuseDuplicateIdentities_NamelessValueIsSkipped(t *testing.T) {
	out := &kernel.RenderResult{Compiled: []*kernel.Compiled{
		compiledObject(t, "nameless", `{apiVersion: "v1", kind: "ConfigMap"}`),
		compiledObject(t, "registration", registrationObject),
		compiledObject(t, "registration-copy", registrationObject),
	}}

	err := refuseDuplicateIdentities(out)

	var dupErr *objectset.DuplicateIdentitiesError
	require.ErrorAs(t, err, &dupErr)
	require.Len(t, dupErr.Duplicates, 1)
	assert.Equal(t, "backup-system.k8up", dupErr.Duplicates[0].Identity.Name)
}

// A render with no compiled objects carries no duplicate.
func TestRefuseDuplicateIdentities_EmptyRender(t *testing.T) {
	assert.NoError(t, refuseDuplicateIdentities(&kernel.RenderResult{}))
}

// The guard compares only an override (flag or env) with the file's
// namespace; a namespace from config or the default, and an env value the
// flag shadows, are never compared.
func TestRefuseNamespaceOverride(t *testing.T) {
	const declared = "media"
	tests := []struct {
		name    string
		ns      config.ResolvedField
		wantErr []string
	}{
		{name: "flag matches", ns: config.ResolvedField{Value: "media", Source: config.SourceFlag}},
		{name: "flag differs", ns: config.ResolvedField{Value: "staging", Source: config.SourceFlag}, wantErr: []string{"--namespace", `"staging"`, `"media"`, "./jellyfin"}},
		{name: "env differs", ns: config.ResolvedField{Value: "staging", Source: config.SourceEnv}, wantErr: []string{"OPM_NAMESPACE", `"staging"`, `"media"`, "./jellyfin"}},
		{name: "config is no override", ns: config.ResolvedField{Value: "staging", Source: config.SourceConfig}},
		{name: "default is no override", ns: config.ResolvedField{Value: "default", Source: config.SourceDefault}},
		{name: "flag matches while env differs", ns: config.ResolvedField{Value: "media", Source: config.SourceFlag, Shadowed: map[config.Source]string{config.SourceEnv: "staging"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := refuseNamespaceOverride("./jellyfin", declared, tt.ns)
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			var valErr *pkgerrors.ValidationError
			require.True(t, errors.As(err, &valErr))
			for _, want := range tt.wantErr {
				assert.Contains(t, valErr.Message, want)
			}
			assert.Contains(t, valErr.Message, "metadata.namespace")
			assert.Contains(t, valErr.Details, "metadata.namespace")
		})
	}

	t.Run("printed form", func(t *testing.T) {
		err := refuseNamespaceOverride("./jellyfin", declared, config.ResolvedField{Value: "staging", Source: config.SourceFlag})
		logs, details := captureValidationOutput(t, err)
		assert.Contains(t, logs, `render failed: --namespace "staging" disagrees with metadata.namespace "media" in ./jellyfin`)
		assert.NotContains(t, logs, "error=", "the refusal prints as a message, not an escaped field")
		assert.Contains(t, details, `set metadata.namespace: "staging" in the instance file`)
		assert.Contains(t, details, `delete the one in "media"`, "a changed namespace is a new instance, not a move")
		assert.False(t, strings.Contains(details, "0011:"), "no enhancement reference in CLI output")
	})
}

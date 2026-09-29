package platform

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
)

// tempOpmDir returns a config.cue path in a fresh OPM home (the file itself
// need not exist); generated platform modules land in its cache.
func tempOpmDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "config.cue")
}

// platformModuleDir writes a minimal platform module (cue.mod/module.cue and
// platform.cue) into a fresh directory and returns it.
func platformModuleDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, config.WritePlatformModule(dir))
	return dir
}

func clusterGetterReturning(doc *ClusterPlatform, unavailable string, err error) ClusterPlatformGetter {
	return func(context.Context) (*ClusterPlatform, string, error) {
		return doc, unavailable, err
	}
}

// specDoc is a Platform document carrying a spec and no status: the
// solo-cluster shape, before any operator has generated for it.
func specDoc(spec map[string]any) *ClusterPlatform {
	return &ClusterPlatform{Name: "cluster", Generation: 1, Spec: spec}
}

// statusRow is one enabled status.registry row as the operator writes it.
// The disabled shape (enabled omitted) is covered in spec_test.go.
func statusRow(catalog, version, source string) map[string]any {
	return map[string]any{"catalog": catalog, "version": version, "enabled": true, "source": source}
}

// captureWarnings redirects the CLI's log sink for the duration of the test
// and returns the buffer the warnings land in.
func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	output.SetLogWriter(&buf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })
	return &buf
}

func TestResolve_FlagWinsOverEverything(t *testing.T) {
	configPath := tempOpmDir(t)
	flagDir := platformModuleDir(t)

	clusterCalled := false
	getter := func(context.Context) (*ClusterPlatform, string, error) {
		clusterCalled = true
		return specDoc(map[string]any{"type": "kubernetes"}), "", nil
	}

	dir, res, err := Resolve(context.Background(), ResolveOptions{
		PlatformFlag: flagDir,
		ConfigPath:   configPath,
		Cluster:      getter,
	})
	require.NoError(t, err)
	assert.Equal(t, flagDir, dir)
	assert.Equal(t, SourceFlagDir, res.Source)
	assert.Equal(t, flagDir, res.Location)
	assert.Equal(t, flagDir, res.Dir)
	assert.False(t, clusterCalled, "flag override must not read the cluster")
}

func TestResolve_ArgumentWinsOverFlagAndEverything(t *testing.T) {
	configPath := tempOpmDir(t)
	argDir := platformModuleDir(t)
	flagDir := platformModuleDir(t)

	clusterCalled := false
	getter := func(context.Context) (*ClusterPlatform, string, error) {
		clusterCalled = true
		return specDoc(map[string]any{"type": "kubernetes"}), "", nil
	}

	dir, res, err := Resolve(context.Background(), ResolveOptions{
		Argument:     argDir,
		PlatformFlag: flagDir,
		ConfigPath:   configPath,
		Cluster:      getter,
	})
	require.NoError(t, err)
	assert.Equal(t, argDir, dir)
	assert.Equal(t, SourceArgumentDir, res.Source)
	assert.Equal(t, argDir, res.Location)
	assert.Equal(t, argDir, res.Dir)
	assert.False(t, clusterCalled, "a directory argument must not read the cluster")
}

// A directory argument gets the same module-shape check the flag gets, so a
// non-module directory is refused before anything is built.
func TestResolve_ArgumentDirWithoutModuleRefused(t *testing.T) {
	configPath := tempOpmDir(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte("package platform\n"), 0o600))

	_, _, err := Resolve(context.Background(), ResolveOptions{Argument: dir, ConfigPath: configPath})
	require.Error(t, err)
	assert.Contains(t, err.Error(), dir)
	assert.Contains(t, err.Error(), "not a platform module")
	assert.Contains(t, err.Error(), "opm platform pull")
}

func TestResolve_FlagFileRefused(t *testing.T) {
	configPath := tempOpmDir(t)
	file := filepath.Join(t.TempDir(), "platform.cue")
	require.NoError(t, os.WriteFile(file, []byte(`name: "x"`), 0o600))

	_, _, err := Resolve(context.Background(), ResolveOptions{PlatformFlag: file, ConfigPath: configPath})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is a file")
	assert.Contains(t, err.Error(), "cue.mod/module.cue")
	assert.Contains(t, err.Error(), "opm platform pull")
}

func TestResolve_FlagDirWithoutModuleRefused(t *testing.T) {
	configPath := tempOpmDir(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte("package platform\n"), 0o600))

	_, _, err := Resolve(context.Background(), ResolveOptions{PlatformFlag: dir, ConfigPath: configPath})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a platform module")
	assert.Contains(t, err.Error(), "opm platform pull")
}

func TestResolve_FlagMissingRefused(t *testing.T) {
	configPath := tempOpmDir(t)

	_, _, err := Resolve(context.Background(), ResolveOptions{PlatformFlag: filepath.Join(t.TempDir(), "nope"), ConfigPath: configPath})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestResolve_ClusterGeneratesModuleUnderCache(t *testing.T) {
	configPath := tempOpmDir(t)
	src := fixtureGraph()

	dir, res, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster: clusterGetterReturning(specDoc(map[string]any{
			"type": "kubernetes",
			"registry": map[string]any{
				"opmodel.dev/catalogs/opm@v4": map[string]any{"version": "4.0.1"},
			},
			"skewPolicy": "Refuse",
		}), "", nil),
		ModFiles: src,
	})
	require.NoError(t, err)
	assert.Equal(t, SourceClusterCR, res.Source)
	assert.Equal(t, "cluster", res.Location)
	assert.Equal(t, dir, res.Dir)
	assert.Equal(t, "Refuse", res.SkewPolicy)
	assert.Equal(t, RegistryOriginSpec, res.RegistryOrigin, "no status: the spec is the registry")
	assert.Empty(t, res.PackageIdentity)
	assert.Empty(t, res.Warning)
	assert.Equal(t, config.PlatformCacheDir(configPath), filepath.Dir(dir), "the generated module lives in the OPM home cache")
	_, err = os.Stat(filepath.Join(dir, "cue.mod", "module.cue"))
	assert.NoError(t, err)
	assert.Contains(t, res.Describe(), dir)
}

func TestResolve_LegacyClusterCRFailsAtGeneration(t *testing.T) {
	configPath := tempOpmDir(t)
	src := fixtureGraph()

	_, _, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster: clusterGetterReturning(specDoc(map[string]any{
			"type": "kubernetes",
			"registry": map[string]any{
				"opmodel.dev/catalogs/opm": map[string]any{"filter": map[string]any{"range": ">=1.0.0-0"}},
			},
		}), "", nil),
		ModFiles: src,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEntryMissingVersion)
	assert.Contains(t, err.Error(), "predate the scalar-version subscription shape")
	assert.Empty(t, src.calls, "no registry access before the legacy-CR refusal")
}

func TestResolve_ClusterHardErrorIsFatal(t *testing.T) {
	configPath := tempOpmDir(t)

	boom := errors.New("connection refused")
	_, _, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster:    clusterGetterReturning(nil, "", boom),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

func TestResolution_Describe(t *testing.T) {
	assert.Equal(t, "platform: /p (argument)", Resolution{Source: SourceArgumentDir, Location: "/p", Dir: "/p"}.Describe())
	assert.Equal(t, "platform: /p (--platform)", Resolution{Source: SourceFlagDir, Location: "/p", Dir: "/p"}.Describe())
	assert.Equal(t, "platform: cluster Platform CR cluster (generated module /c/abc)",
		Resolution{Source: SourceClusterCR, Location: "cluster", Dir: "/c/abc"}.Describe())
	assert.Equal(t, "platform: cluster Platform CR cluster (effective registry, package gen-7-3f9a1c2b, generated module /c/abc)",
		Resolution{Source: SourceClusterCR, Location: "cluster", Dir: "/c/abc",
			RegistryOrigin: RegistryOriginEffective, PackageIdentity: "gen-7-3f9a1c2b"}.Describe())
	assert.Equal(t, "platform: cluster Platform CR cluster (effective registry, no package identity recorded, generated module /c/abc)",
		Resolution{Source: SourceClusterCR, Location: "cluster", Dir: "/c/abc",
			RegistryOrigin: RegistryOriginEffective}.Describe())
	assert.Equal(t, "platform: cluster Platform CR cluster (spec registry, no operator generation recorded, generated module /c/abc)",
		Resolution{Source: SourceClusterCR, Location: "cluster", Dir: "/c/abc",
			RegistryOrigin: RegistryOriginSpec}.Describe())
	assert.Equal(t, "platform: module deps (opmodel.dev/catalogs/k8s@v1 v1.0.0, opmodel.dev/catalogs/opm@v4 v4.4.0; generated module /c/abc)",
		Resolution{Source: SourceModuleDeps, DepsKind: DepsModule, Dir: "/c/abc",
			Catalogs: []string{"opmodel.dev/catalogs/k8s@v1 v1.0.0", "opmodel.dev/catalogs/opm@v4 v4.4.0"}}.Describe())
	assert.Equal(t, "platform: module deps (no catalogs; generated module /c/abc)",
		Resolution{Source: SourceModuleDeps, DepsKind: DepsModule, Dir: "/c/abc"}.Describe())
	assert.Equal(t, "platform: instance deps (opmodel.dev/catalogs/opm@v4 v4.4.0; generated module /c/abc)",
		Resolution{Source: SourceModuleDeps, DepsKind: DepsInstance, Dir: "/c/abc",
			Catalogs: []string{"opmodel.dev/catalogs/opm@v4 v4.4.0"}}.Describe())
}

// moduleDepsOpts is ResolveOptions for a module render with no --platform:
// the module pins one catalog and an older core.
func moduleDepsOpts(t *testing.T, configPath string) ResolveOptions {
	t.Helper()
	return ResolveOptions{
		ConfigPath: configPath,
		DepsKind:   DepsModule,
		Deps: &ModuleDeps{
			ModFile: moduleFileWith(t, map[string]string{
				"opmodel.dev/catalogs/opm@v4": "v4.0.1",
				"opmodel.dev/core@v2":         "v2.0.0-alpha.6",
			}),
			ModFileName: "/m/cue.mod/module.cue",
		},
		ModFiles: depsGraph(),
	}
}

func TestResolve_ModuleDepsGeneratesUnderCache(t *testing.T) {
	configPath := tempOpmDir(t)

	dir, res, err := Resolve(context.Background(), moduleDepsOpts(t, configPath))
	require.NoError(t, err)

	assert.Equal(t, SourceModuleDeps, res.Source)
	assert.Equal(t, config.PlatformCacheDir(configPath), filepath.Dir(dir))
	assert.Equal(t, dir, res.Dir)
	assert.Equal(t, []string{"opmodel.dev/catalogs/opm@v4 v4.0.1"}, res.Catalogs)
	assert.Empty(t, res.Carried)
	assert.Empty(t, res.Warning)
	assert.Equal(t, "platform: module deps (opmodel.dev/catalogs/opm@v4 v4.0.1; generated module "+dir+")", res.Describe())
	_, err = os.Stat(config.PlatformDir(configPath))
	assert.ErrorIs(t, err, os.ErrNotExist, "resolution never seeds or reads a platform in the OPM home")
}

func TestResolve_ModuleDepsFlagWins(t *testing.T) {
	flagDir := platformModuleDir(t)
	opts := moduleDepsOpts(t, tempOpmDir(t))
	opts.PlatformFlag = flagDir
	graph := depsGraph()
	opts.ModFiles = graph

	dir, res, err := Resolve(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, SourceFlagDir, res.Source)
	assert.Equal(t, flagDir, dir)
	assert.Empty(t, graph.calls, "no platform is generated from the deps when the flag is set")
}

func TestResolve_ModuleDepsUnpublishedPinFails(t *testing.T) {
	opts := moduleDepsOpts(t, tempOpmDir(t))
	opts.Deps.ModFile = moduleFileWith(t, map[string]string{"opmodel.dev/catalogs/opm@v4": "v4.9.9"})

	_, _, err := Resolve(context.Background(), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opmodel.dev/catalogs/opm@v4.9.9")
}

// The effective registry the operator recorded is what the cluster renders
// against, so it wins over the authored spec: both entries reach the
// generated module, including the catalog only a registration contributed.
func TestResolve_EffectiveRegistryWinsOverSpec(t *testing.T) {
	configPath := tempOpmDir(t)
	src := fixtureGraph()

	dir, res, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster: clusterGetterReturning(&ClusterPlatform{
			Name:       "cluster",
			Generation: 7,
			Spec: map[string]any{
				"type": "kubernetes",
				"registry": map[string]any{
					"opmodel.dev/catalogs/opm@v4": map[string]any{"version": "4.0.1"},
				},
			},
			Status: map[string]any{
				"observedGeneration": int64(7),
				"packageIdentity":    "gen-7-3f9a1c2b",
				"operatorVersion":    "v1.0.0-alpha.20",
				"registry": []any{
					statusRow("opmodel.dev/catalogs/opm@v4", "4.0.1", EntrySourceSubscription),
					statusRow("opmodel.dev/catalogs/k8s@v1", "1.0.0-alpha.2", EntrySourceRegistration),
				},
				"conditions": []any{
					map[string]any{"type": "Ready", "status": "True", "reason": "Generated"},
				},
			},
		}, "", nil),
		ModFiles: src,
	})
	require.NoError(t, err)
	assert.Equal(t, RegistryOriginEffective, res.RegistryOrigin)
	assert.Equal(t, "gen-7-3f9a1c2b", res.PackageIdentity)
	assert.Contains(t, res.Describe(), "effective registry, package gen-7-3f9a1c2b")

	modFile, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(modFile), "opmodel.dev/catalogs/opm@v4",
		"the subscribed catalog is pinned")
	assert.Contains(t, string(modFile), "opmodel.dev/catalogs/k8s@v1",
		"the registration-sourced catalog is pinned too")
}

// A status behind the spec's generation is warned, never silently
// substituted: the effective package is still what the cluster renders.
func TestResolve_StaleStatusWarns(t *testing.T) {
	configPath := tempOpmDir(t)
	buf := captureWarnings(t)

	_, res, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster: clusterGetterReturning(&ClusterPlatform{
			Name:       "cluster",
			Generation: 5,
			Spec:       map[string]any{"type": "kubernetes"},
			Status: map[string]any{
				"observedGeneration": int64(4),
				"registry": []any{
					statusRow("opmodel.dev/catalogs/opm@v4", "4.0.1", EntrySourceSubscription),
				},
			},
		}, "", nil),
		ModFiles: fixtureGraph(),
	})
	require.NoError(t, err)
	assert.Equal(t, RegistryOriginEffective, res.RegistryOrigin,
		"a stale status does not change which registry is used")

	emitted := buf.String()
	assert.Contains(t, emitted, "generation 5 is not yet generated by the operator")
	assert.Contains(t, emitted, "status describes generation 4")
}

// A refused Platform renders against the last good package the operator
// recorded, and the warning names the operator's reason.
func TestResolve_NotReadyPlatformWarns(t *testing.T) {
	configPath := tempOpmDir(t)
	buf := captureWarnings(t)

	_, res, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster: clusterGetterReturning(&ClusterPlatform{
			Name:       "cluster",
			Generation: 4,
			Spec:       map[string]any{"type": "kubernetes"},
			Status: map[string]any{
				"observedGeneration": int64(4),
				"registry": []any{
					statusRow("opmodel.dev/catalogs/opm@v4", "4.0.1", EntrySourceSubscription),
				},
				"conditions": []any{
					map[string]any{"type": "Ready", "status": "False", "reason": "OverSubscribedContracts"},
				},
			},
		}, "", nil),
		ModFiles: fixtureGraph(),
	})
	require.NoError(t, err)
	assert.Equal(t, RegistryOriginEffective, res.RegistryOrigin)

	emitted := buf.String()
	assert.Contains(t, emitted, "Ready=False (OverSubscribedContracts)")
	assert.Contains(t, emitted, "last good package")
}

// A status carrying no registry is the pre-0015 operator (or none at all):
// the spec is generated from and the provenance says why.
func TestResolve_StatusWithoutRegistryFallsBackToSpec(t *testing.T) {
	configPath := tempOpmDir(t)

	_, res, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster: clusterGetterReturning(&ClusterPlatform{
			Name:       "cluster",
			Generation: 2,
			Spec: map[string]any{
				"type": "kubernetes",
				"registry": map[string]any{
					"opmodel.dev/catalogs/opm@v4": map[string]any{"version": "4.0.1"},
				},
			},
			Status: map[string]any{"operatorVersion": "v0.9.0"},
		}, "", nil),
		ModFiles: fixtureGraph(),
	})
	require.NoError(t, err)
	assert.Equal(t, RegistryOriginSpec, res.RegistryOrigin)
	assert.Empty(t, res.PackageIdentity)
	assert.Contains(t, res.Describe(), "spec registry, no operator generation recorded")
}

// A hand-written status row without a version is refused before generation.
func TestResolve_StatusEntryWithoutVersionRefused(t *testing.T) {
	configPath := tempOpmDir(t)
	src := fixtureGraph()

	_, _, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster: clusterGetterReturning(&ClusterPlatform{
			Name:       "cluster",
			Generation: 1,
			Spec:       map[string]any{"type": "kubernetes"},
			Status: map[string]any{
				"registry": []any{
					map[string]any{"catalog": "opmodel.dev/catalogs/opm@v4", "source": EntrySourceSubscription},
				},
			},
		}, "", nil),
		ModFiles: src,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no version")
	assert.Empty(t, src.calls, "no registry access before the refusal")
}

// A fatal read is distinguishable from an absent CR, so a caller can map it
// onto the connectivity exit code.
func TestResolve_ClusterHardErrorCarriesTheReadSentinel(t *testing.T) {
	configPath := tempOpmDir(t)

	boom := errors.New("connection refused")
	_, _, err := Resolve(context.Background(), ResolveOptions{
		ConfigPath: configPath,
		Cluster:    clusterGetterReturning(nil, "", boom),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrClusterRead)
	assert.NotErrorIs(t, err, ErrNoClusterPlatform)
}

// clusterStepOutcomes are the three getter results of design.md's
// cluster-step table: a readable Platform, an absent or forbidden one, and
// any other failure.
var clusterStepOutcomes = map[string]func() ClusterPlatformGetter{
	"read": func() ClusterPlatformGetter {
		return clusterGetterReturning(specDoc(map[string]any{
			"type":     "kubernetes",
			"registry": map[string]any{"opmodel.dev/catalogs/opm@v4": map[string]any{"version": "4.0.1"}},
		}), "", nil)
	},
	"absent": func() ClusterPlatformGetter {
		return clusterGetterReturning(nil, "no Platform CR in the cluster", nil)
	},
	"unreachable": func() ClusterPlatformGetter {
		return clusterGetterReturning(nil, "", errors.New("dial tcp 192.0.2.1:6443: i/o timeout"))
	},
}

// TestResolve_ClusterStep covers every cell of the cluster-step table: the
// getter result against the default, ClusterOptional and NoFallback modes.
// Deps are set in every mode but NoFallback, as the commands set them.
func TestResolve_ClusterStep(t *testing.T) {
	type want struct {
		source   Source
		errIs    error
		warnings []string
	}
	deps := func(w ...string) want { return want{source: SourceModuleDeps, warnings: w} }
	tests := []struct {
		outcome string
		mode    string
		want    want
	}{
		{"read", "default", want{source: SourceClusterCR}},
		{"read", "optional", want{source: SourceClusterCR}},
		{"read", "nofallback", want{source: SourceClusterCR}},
		{"absent", "default", deps("cluster Platform not used (no Platform CR in the cluster)", "rendering against the instance's own deps")},
		{"absent", "optional", deps("cluster Platform not used (no Platform CR in the cluster)", "rendering against the instance's own deps")},
		{"absent", "nofallback", want{errIs: ErrNoClusterPlatform}},
		{"unreachable", "default", want{errIs: ErrClusterRead}},
		{"unreachable", "optional", deps("cluster Platform not used (could not reach the cluster: dial tcp 192.0.2.1:6443: i/o timeout)", "rendering against the instance's own deps")},
		{"unreachable", "nofallback", want{errIs: ErrClusterRead}},
	}
	for _, tt := range tests {
		t.Run(tt.outcome+"/"+tt.mode, func(t *testing.T) {
			buf := captureWarnings(t)
			opts := moduleDepsOpts(t, tempOpmDir(t))
			opts.DepsKind = DepsInstance
			opts.Cluster = clusterStepOutcomes[tt.outcome]()
			switch tt.mode {
			case "optional":
				opts.ClusterOptional = true
			case "nofallback":
				opts.NoFallback = true
				opts.Deps = nil
			}
			opts.ModFiles = depsGraph()

			_, res, err := Resolve(context.Background(), opts)
			if tt.want.errIs != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.want.errIs)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.source, res.Source)
			for _, w := range tt.want.warnings {
				assert.Contains(t, buf.String(), w)
				assert.Contains(t, res.Warning, w)
			}
			if len(tt.want.warnings) == 0 {
				assert.Empty(t, res.Warning)
				assert.NotContains(t, buf.String(), "cluster Platform not used")
			}
		})
	}
}

// The module-deps fallback names the module in its warning.
func TestResolve_ModuleDepsFallbackWarningNamesTheModule(t *testing.T) {
	buf := captureWarnings(t)
	opts := moduleDepsOpts(t, tempOpmDir(t))
	opts.Cluster = clusterStepOutcomes["absent"]()

	_, res, err := Resolve(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, DepsModule, res.DepsKind)
	assert.Contains(t, buf.String(), "rendering against the module's own deps")
}

// No cluster getter (offline, or no kubeconfig context) resolves the deps
// silently: nothing fell back.
func TestResolve_NoClusterResolvesDepsWithoutWarning(t *testing.T) {
	buf := captureWarnings(t)
	opts := moduleDepsOpts(t, tempOpmDir(t))
	opts.DepsKind = DepsInstance

	_, res, err := Resolve(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, SourceModuleDeps, res.Source)
	assert.Equal(t, DepsInstance, res.DepsKind)
	assert.Empty(t, res.Warning)
	assert.Empty(t, buf.String())
}

// With no step set there is nothing to resolve, and the error names the
// sources that were looked for.
func TestResolve_NoSourceAvailable(t *testing.T) {
	_, _, err := Resolve(context.Background(), ResolveOptions{ConfigPath: tempOpmDir(t)})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoPlatformSource)
	assert.Contains(t, err.Error(), "--platform")
	assert.Contains(t, err.Error(), "cluster Platform")

	_, _, err = Resolve(context.Background(), ResolveOptions{
		ConfigPath: tempOpmDir(t),
		Cluster:    clusterStepOutcomes["absent"](),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoPlatformSource)
	assert.Contains(t, err.Error(), "no Platform CR in the cluster")
}

// NoFallback without a cluster getter refuses: the command's subject is a
// cluster Platform and there is no cluster to read it from.
func TestResolve_NoFallbackWithoutClusterRefuses(t *testing.T) {
	_, _, err := Resolve(context.Background(), ResolveOptions{ConfigPath: tempOpmDir(t), NoFallback: true})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoPlatformSource)
}

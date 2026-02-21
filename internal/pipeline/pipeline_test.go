package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTempValues writes a CUE string to a temp file and returns its path.
func writeTempValues(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "values-*.cue")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

// concreteValues returns a temp values file path with concrete defaults for real_module.
// real_module's #config requires image (string, no default) and replicas (int >=1, default 1).
func concreteValues(t *testing.T) string {
	t.Helper()
	return writeTempValues(t, "values: {\n\timage:    \"nginx:latest\"\n\treplicas: 1\n}\n")
}

// requireRegistry skips the test if OPM_REGISTRY is not set and configures
// CUE_REGISTRY for the duration of the test.
func requireRegistry(t *testing.T) {
	t.Helper()
	registry := os.Getenv("OPM_REGISTRY")
	if registry == "" {
		t.Skip("OPM_REGISTRY not set — skipping registry-dependent test")
	}
	t.Setenv("CUE_REGISTRY", registry)
}

// realModulePath returns the absolute path to the real_module test fixture
// that imports opmodel.dev/core@v0 (required for Approach C builder).
func realModulePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	// internal/pipeline/ → internal/ → cli/ → experiments/module-release-cue-eval/testdata/real_module
	cliRoot := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(cliRoot, "experiments", "module-release-cue-eval", "testdata", "real_module")
}

// buildMatchingProvider compiles a CUE provider value with a WebTransformer that
// matches real_module's "web" component (requires the Container resource).
// The transformer does NOT declare the Scaling trait, so it will be unhandled.
func buildMatchingProvider(t *testing.T, cueCtx *cue.Context) cue.Value {
	t.Helper()
	v := cueCtx.CompileString(`{
		version: "1.0.0"
		transformers: {
			WebTransformer: {
				requiredResources: { "opmodel.dev/resources/workload@v0#Container": _ }
				#transform: {
					#component: _
					#context: { name: string, namespace: string, #moduleReleaseMetadata: _, #componentMetadata: _ }
					output: {
						apiVersion: "apps/v1"
						kind: "Deployment"
						metadata: { name: #context.name, namespace: #context.namespace }
						spec: {}
					}
				}
			}
		}
	}`)
	require.NoError(t, v.Err())
	return v
}

// buildBrokenProvider compiles a provider whose transformer matches the web
// component but has no #transform field, causing a TransformError on Execute().
func buildBrokenProvider(t *testing.T, cueCtx *cue.Context) cue.Value {
	t.Helper()
	v := cueCtx.CompileString(`{
		version: "1.0.0"
		transformers: {
			BrokenTransformer: {
				requiredResources: { "opmodel.dev/resources/workload@v0#Container": _ }
			}
		}
	}`)
	require.NoError(t, v.Err())
	return v
}

// TestNewPipeline verifies the constructor returns a non-nil Pipeline.
func TestNewPipeline(t *testing.T) {
	p := NewPipeline(nil, nil, "")
	assert.NotNil(t, p)
}

// TestRenderOptionsValidate verifies option validation.
func TestRenderOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    RenderOptions
		wantErr bool
	}{
		{
			name:    "valid options",
			opts:    RenderOptions{ModulePath: "/some/path"},
			wantErr: false,
		},
		{
			name:    "missing module path",
			opts:    RenderOptions{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPipeline_LoaderFailure_FatalError verifies that a non-existent module path
// returns a fatal error with nil RenderResult; no downstream phases are called.
func TestPipeline_LoaderFailure_FatalError(t *testing.T) {
	ctx := context.Background()
	cueCtx := cuecontext.New()
	p := NewPipeline(cueCtx, nil, "")

	result, err := p.Render(ctx, RenderOptions{
		ModulePath: "/nonexistent/module/path",
		Namespace:  "default",
	})

	assert.Error(t, err, "loader failure should return a fatal error")
	assert.Nil(t, result, "RenderResult should be nil on fatal error")
}

// TestPipeline_ProviderFailure_FatalError verifies that a nil/empty providers map
// returns a fatal error (provider.Load fails before BUILD phase).
func TestPipeline_ProviderFailure_FatalError(t *testing.T) {
	ctx := context.Background()
	cueCtx := cuecontext.New()

	modPath, err := filepath.Abs("testdata/test-module")
	require.NoError(t, err)

	// nil providers → provider.Load returns "no providers configured"
	p := NewPipeline(cueCtx, nil, "")

	result, err := p.Render(ctx, RenderOptions{
		ModulePath: modPath,
		Namespace:  "default",
	})

	assert.Error(t, err, "provider failure should return a fatal error")
	assert.Nil(t, result, "RenderResult should be nil on provider failure")
}

// TestPipeline_SuccessfulRender verifies end-to-end render produces a non-nil
// RenderResult with resources and no errors. Requires OPM_REGISTRY.
func TestPipeline_SuccessfulRender(t *testing.T) {
	requireRegistry(t)

	ctx := context.Background()
	cueCtx := cuecontext.New()
	registry := os.Getenv("OPM_REGISTRY")

	providers := map[string]cue.Value{"test": buildMatchingProvider(t, cueCtx)}
	p := NewPipeline(cueCtx, providers, registry)

	result, err := p.Render(ctx, RenderOptions{
		ModulePath: realModulePath(t),
		Name:       "test-release",
		Namespace:  "default",
		Provider:   "test",
		Values:     []string{concreteValues(t)},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Resources, "should produce at least one resource")
	assert.Equal(t, "test-release", result.Release.Name)
	assert.Equal(t, "default", result.Release.Namespace)
}

// TestPipeline_RenderErrors_InResult verifies that errors from transformer
// execution land in RenderResult.Errors while Render() itself returns nil.
// Uses a transformer without #transform to trigger a TransformError.
// Requires OPM_REGISTRY.
func TestPipeline_RenderErrors_InResult(t *testing.T) {
	requireRegistry(t)

	ctx := context.Background()
	cueCtx := cuecontext.New()
	registry := os.Getenv("OPM_REGISTRY")

	providers := map[string]cue.Value{"test": buildBrokenProvider(t, cueCtx)}
	p := NewPipeline(cueCtx, providers, registry)

	result, err := p.Render(ctx, RenderOptions{
		ModulePath: realModulePath(t),
		Name:       "test-release",
		Namespace:  "default",
		Provider:   "test",
		Values:     []string{concreteValues(t)},
	})

	assert.NoError(t, err, "transform errors are render errors, not fatal")
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Errors, "RenderResult.Errors should contain the transform error")
}

// TestPipeline_UnhandledTrait_NonStrict_Warning verifies that an unhandled trait
// produces a warning (not an error) when Strict is false. Requires OPM_REGISTRY.
func TestPipeline_UnhandledTrait_NonStrict_Warning(t *testing.T) {
	requireRegistry(t)

	ctx := context.Background()
	cueCtx := cuecontext.New()
	registry := os.Getenv("OPM_REGISTRY")

	// WebTransformer matches by resource but doesn't declare the Scaling trait
	// → Scaling will be unhandled → warning in non-strict mode.
	providers := map[string]cue.Value{"test": buildMatchingProvider(t, cueCtx)}
	p := NewPipeline(cueCtx, providers, registry)

	result, err := p.Render(ctx, RenderOptions{
		ModulePath: realModulePath(t),
		Name:       "test-release",
		Namespace:  "default",
		Provider:   "test",
		Strict:     false,
		Values:     []string{concreteValues(t)},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Warnings, "unhandled Scaling trait should produce a warning in non-strict mode")
	for _, e := range result.Errors {
		assert.NotContains(t, e.Error(), "unhandled trait",
			"unhandled trait should not appear in Errors in non-strict mode")
	}
}

// TestPipeline_UnhandledTrait_Strict_Error verifies that an unhandled trait
// produces an error (not a warning) when Strict is true. Requires OPM_REGISTRY.
func TestPipeline_UnhandledTrait_Strict_Error(t *testing.T) {
	requireRegistry(t)

	ctx := context.Background()
	cueCtx := cuecontext.New()
	registry := os.Getenv("OPM_REGISTRY")

	providers := map[string]cue.Value{"test": buildMatchingProvider(t, cueCtx)}
	p := NewPipeline(cueCtx, providers, registry)

	result, err := p.Render(ctx, RenderOptions{
		ModulePath: realModulePath(t),
		Name:       "test-release",
		Namespace:  "default",
		Provider:   "test",
		Strict:     true,
		Values:     []string{concreteValues(t)},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Warnings, "warnings should be empty when strict=true")
	assert.NotEmpty(t, result.Errors, "unhandled Scaling trait should appear in Errors in strict mode")
}

// TestPipeline_ContextCancellation_FatalError verifies that context cancellation
// during GENERATE returns a fatal error with nil RenderResult.
// Requires OPM_REGISTRY.
func TestPipeline_ContextCancellation_FatalError(t *testing.T) {
	requireRegistry(t)

	cueCtx := cuecontext.New()
	registry := os.Getenv("OPM_REGISTRY")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately so Execute sees a done context on its first check.

	providers := map[string]cue.Value{"test": buildMatchingProvider(t, cueCtx)}
	p := NewPipeline(cueCtx, providers, registry)

	result, err := p.Render(ctx, RenderOptions{
		ModulePath: realModulePath(t),
		Name:       "test-release",
		Namespace:  "default",
		Provider:   "test",
		Values:     []string{concreteValues(t)},
	})

	assert.Error(t, err, "canceled context should return a fatal error")
	assert.Nil(t, result, "RenderResult should be nil on context cancellation")
	assert.ErrorIs(t, err, context.Canceled)
}

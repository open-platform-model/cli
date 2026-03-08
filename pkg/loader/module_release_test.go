package loader

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetectReleaseKind tests kind detection from an in-memory CUE value.
func TestDetectReleaseKind(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name      string
		cue       string
		wantKind  string
		wantErr   bool
		errSubstr string
	}{
		{
			name:     "ModuleRelease",
			cue:      `{ kind: "ModuleRelease" }`,
			wantKind: "ModuleRelease",
		},
		{
			name:     "BundleRelease",
			cue:      `{ kind: "BundleRelease" }`,
			wantKind: "BundleRelease",
		},
		{
			name:      "unknown kind",
			cue:       `{ kind: "FooBar" }`,
			wantErr:   true,
			errSubstr: "unknown release kind",
		},
		{
			name:      "missing kind",
			cue:       `{ name: "test" }`,
			wantErr:   true,
			errSubstr: "no 'kind' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := ctx.CompileString(tt.cue)
			require.NoError(t, v.Err())

			kind, err := DetectReleaseKind(v)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantKind, kind)
			}
		})
	}
}

// TestResolveReleaseFile tests directory detection via os.Stat (DEBT #10).
func TestResolveReleaseFile(t *testing.T) {
	// Create a temp directory to test directory detection.
	tmpDir := t.TempDir()

	// Create a release.cue file inside the temp dir.
	releasePath := filepath.Join(tmpDir, "release.cue")
	require.NoError(t, os.WriteFile(releasePath, []byte(`kind: "ModuleRelease"`), 0o644))

	t.Run("directory resolves to release.cue", func(t *testing.T) {
		got, err := resolveReleaseFile(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, releasePath, got)
	})

	t.Run("file path returned as-is", func(t *testing.T) {
		got, err := resolveReleaseFile(releasePath)
		require.NoError(t, err)
		assert.Equal(t, releasePath, got)
	})

	t.Run("empty path returns error", func(t *testing.T) {
		_, err := resolveReleaseFile("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be empty")
	})

	t.Run("non-existent path returned as-is", func(t *testing.T) {
		nonExistent := filepath.Join(tmpDir, "doesnotexist.cue")
		got, err := resolveReleaseFile(nonExistent)
		require.NoError(t, err)
		assert.Equal(t, nonExistent, got)
	})
}

// TestFinalizeValue tests constraint stripping via finalizeValue.
func TestFinalizeValue(t *testing.T) {
	ctx := cuecontext.New()

	// A concrete value with no schema constraints.
	v := ctx.CompileString(`{
		apiVersion: "apps/v1"
		kind:       "Deployment"
		metadata: name: "my-app"
	}`)
	require.NoError(t, v.Err())

	out, err := finalizeValue(ctx, v)
	require.NoError(t, err)
	assert.NoError(t, out.Err())

	// The finalized value should still contain the same data.
	name, err := out.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "my-app", name)
}

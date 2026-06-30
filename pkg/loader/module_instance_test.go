package loader

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetectInstanceKind tests kind detection from an in-memory CUE value.
func TestDetectInstanceKind(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name      string
		cue       string
		wantKind  string
		wantErr   bool
		errSubstr string
	}{
		{
			name:     "ModuleInstance",
			cue:      `{ kind: "ModuleInstance" }`,
			wantKind: "ModuleInstance",
		},
		{
			// BundleRelease is no longer recognized — the bundle path was
			// removed in 0002 X2 (D15, supersedes D7).
			name:      "BundleRelease rejected",
			cue:       `{ kind: "BundleRelease" }`,
			wantErr:   true,
			errSubstr: "unknown instance kind",
		},
		{
			name:      "unknown kind",
			cue:       `{ kind: "FooBar" }`,
			wantErr:   true,
			errSubstr: "unknown instance kind",
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

			kind, err := DetectInstanceKind(v)
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

// TestResolveInstanceFile tests directory detection via os.Stat (DEBT #10).
func TestResolveInstanceFile(t *testing.T) {
	// Create a temp directory to test directory detection.
	tmpDir := t.TempDir()

	// Create an instance.cue file inside the temp dir.
	instancePath := filepath.Join(tmpDir, "instance.cue")
	require.NoError(t, os.WriteFile(instancePath, []byte(`kind: "ModuleInstance"`), 0o644))

	t.Run("directory resolves to instance.cue", func(t *testing.T) {
		got, err := resolveInstanceFile(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, instancePath, got)
	})

	t.Run("file path returned as-is", func(t *testing.T) {
		got, err := resolveInstanceFile(instancePath)
		require.NoError(t, err)
		assert.Equal(t, instancePath, got)
	})

	t.Run("empty path returns error", func(t *testing.T) {
		_, err := resolveInstanceFile("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be empty")
	})

	t.Run("non-existent path returns error", func(t *testing.T) {
		nonExistent := filepath.Join(tmpDir, "doesnotexist.cue")
		_, err := resolveInstanceFile(nonExistent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("directory without instance file returns error", func(t *testing.T) {
		emptyDir := t.TempDir()
		_, err := resolveInstanceFile(emptyDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not contain instance.cue")
	})
}

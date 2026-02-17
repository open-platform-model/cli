package cmdutil

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderFlags_AddTo(t *testing.T) {
	var rf RenderFlags
	cmd := &cobra.Command{Use: "test"}
	rf.AddTo(cmd)

	// Verify flags exist with correct names and types
	valuesFlag := cmd.Flags().Lookup("values")
	require.NotNil(t, valuesFlag)
	assert.Equal(t, "f", valuesFlag.Shorthand)
	assert.Equal(t, "stringArray", valuesFlag.Value.Type())

	nsFlag := cmd.Flags().Lookup("namespace")
	require.NotNil(t, nsFlag)
	assert.Equal(t, "n", nsFlag.Shorthand)
	assert.Equal(t, "", nsFlag.DefValue)

	rnFlag := cmd.Flags().Lookup("release-name")
	require.NotNil(t, rnFlag)
	assert.Equal(t, "", rnFlag.DefValue)

	provFlag := cmd.Flags().Lookup("provider")
	require.NotNil(t, provFlag)
	assert.Equal(t, "", provFlag.DefValue)
}

func TestK8sFlags_AddTo(t *testing.T) {
	var kf K8sFlags
	cmd := &cobra.Command{Use: "test"}
	kf.AddTo(cmd)

	kcFlag := cmd.Flags().Lookup("kubeconfig")
	require.NotNil(t, kcFlag)
	assert.Equal(t, "", kcFlag.DefValue)

	ctxFlag := cmd.Flags().Lookup("context")
	require.NotNil(t, ctxFlag)
	assert.Equal(t, "", ctxFlag.DefValue)
}

func TestReleaseSelectorFlags_AddTo(t *testing.T) {
	var rsf ReleaseSelectorFlags
	cmd := &cobra.Command{Use: "test"}
	rsf.AddTo(cmd)

	nsFlag := cmd.Flags().Lookup("namespace")
	require.NotNil(t, nsFlag)
	assert.Equal(t, "n", nsFlag.Shorthand)

	rnFlag := cmd.Flags().Lookup("release-name")
	require.NotNil(t, rnFlag)

	ridFlag := cmd.Flags().Lookup("release-id")
	require.NotNil(t, ridFlag)
}

func TestReleaseSelectorFlags_Validate(t *testing.T) {
	tests := []struct {
		name        string
		releaseName string
		releaseID   string
		wantErr     string
	}{
		{
			name:        "both set",
			releaseName: "my-app",
			releaseID:   "abc-123",
			wantErr:     "mutually exclusive",
		},
		{
			name:    "neither set",
			wantErr: "either --release-name or --release-id is required",
		},
		{
			name:        "only release-name",
			releaseName: "my-app",
		},
		{
			name:      "only release-id",
			releaseID: "abc-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsf := ReleaseSelectorFlags{
				ReleaseName: tt.releaseName,
				ReleaseID:   tt.releaseID,
			}
			err := rsf.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestReleaseSelectorFlags_LogName(t *testing.T) {
	tests := []struct {
		name        string
		releaseName string
		releaseID   string
		want        string
	}{
		{
			name:        "release name set",
			releaseName: "my-app",
			releaseID:   "a1b2c3d4-e5f6-7890-abcd",
			want:        "my-app",
		},
		{
			name:      "only release ID",
			releaseID: "a1b2c3d4-e5f6-7890-abcd",
			want:      "release:a1b2c3d4",
		},
		{
			name:      "short release ID",
			releaseID: "abc",
			want:      "release:abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsf := ReleaseSelectorFlags{
				ReleaseName: tt.releaseName,
				ReleaseID:   tt.releaseID,
			}
			assert.Equal(t, tt.want, rsf.LogName())
		})
	}
}

func TestFlagGroupComposition(t *testing.T) {
	// Verify RenderFlags + K8sFlags can coexist on the same command
	var rf RenderFlags
	var kf K8sFlags
	cmd := &cobra.Command{Use: "test"}
	rf.AddTo(cmd)
	kf.AddTo(cmd)

	// All 6 flags should be registered
	expectedFlags := []string{"values", "namespace", "release-name", "provider", "kubeconfig", "context"}
	for _, name := range expectedFlags {
		flag := cmd.Flags().Lookup(name)
		assert.NotNil(t, flag, "flag %q should be registered", name)
	}
}

func TestResolveModulePath(t *testing.T) {
	assert.Equal(t, ".", ResolveModulePath(nil))
	assert.Equal(t, ".", ResolveModulePath([]string{}))
	assert.Equal(t, "./my-module", ResolveModulePath([]string{"./my-module"}))
}

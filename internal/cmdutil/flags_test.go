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

	rnFlag := cmd.Flags().Lookup("instance-name")
	require.NotNil(t, rnFlag)
	assert.Equal(t, "", rnFlag.DefValue)

	require.Nil(t, cmd.Flags().Lookup("provider"), "--provider is retired (0006 D21)")
	platformFlag := cmd.Flags().Lookup("platform")
	require.NotNil(t, platformFlag)
	assert.Equal(t, "", platformFlag.DefValue)
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

func TestInstanceSelectorFlags_AddTo(t *testing.T) {
	var rsf InstanceSelectorFlags
	cmd := &cobra.Command{Use: "test"}
	rsf.AddTo(cmd)

	nsFlag := cmd.Flags().Lookup("namespace")
	require.NotNil(t, nsFlag)
	assert.Equal(t, "n", nsFlag.Shorthand)

	rnFlag := cmd.Flags().Lookup("instance-name")
	require.NotNil(t, rnFlag)

	ridFlag := cmd.Flags().Lookup("instance-id")
	require.NotNil(t, ridFlag)
}

func TestInstanceSelectorFlags_Validate(t *testing.T) {
	tests := []struct {
		name         string
		instanceName string
		instanceID   string
		wantErr      string
	}{
		{
			name:         "both set",
			instanceName: "my-app",
			instanceID:   "abc-123",
			wantErr:      "mutually exclusive",
		},
		{
			name:    "neither set",
			wantErr: "either --instance-name or --instance-id is required",
		},
		{
			name:         "only instance-name",
			instanceName: "my-app",
		},
		{
			name:       "only instance-id",
			instanceID: "abc-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsf := InstanceSelectorFlags{
				InstanceName: tt.instanceName,
				InstanceID:   tt.instanceID,
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

func TestInstanceSelectorFlags_LogName(t *testing.T) {
	tests := []struct {
		name         string
		instanceName string
		instanceID   string
		want         string
	}{
		{
			name:         "instance name set",
			instanceName: "my-app",
			instanceID:   "a1b2c3d4-e5f6-7890-abcd",
			want:         "my-app",
		},
		{
			name:       "only instance ID",
			instanceID: "a1b2c3d4-e5f6-7890-abcd",
			want:       "instance:a1b2c3d4",
		},
		{
			name:       "short instance ID",
			instanceID: "abc",
			want:       "instance:abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsf := InstanceSelectorFlags{
				InstanceName: tt.instanceName,
				InstanceID:   tt.instanceID,
			}
			assert.Equal(t, tt.want, rsf.LogName())
		})
	}
}

func TestInstanceFileFlags_AddTo(t *testing.T) {
	var rff InstanceFileFlags
	cmd := &cobra.Command{Use: "test"}
	rff.AddTo(cmd)

	require.Nil(t, cmd.Flags().Lookup("provider"), "--provider is retired (0006 D21)")
	platformFlag := cmd.Flags().Lookup("platform")
	require.NotNil(t, platformFlag)
	assert.Equal(t, "", platformFlag.DefValue)

	valuesFlag := cmd.Flags().Lookup("values")
	require.NotNil(t, valuesFlag)
	assert.Equal(t, "stringArray", valuesFlag.Value.Type())
}

func TestResolveInstanceIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		wantName string
		wantUUID string
	}{
		{
			name:     "plain instance name",
			arg:      "jellyfin",
			wantName: "jellyfin",
			wantUUID: "",
		},
		{
			name:     "instance name with hyphens",
			arg:      "my-app-prod",
			wantName: "my-app-prod",
			wantUUID: "",
		},
		{
			name:     "valid UUID",
			arg:      "550e8400-e29b-41d4-a716-446655440000",
			wantName: "",
			wantUUID: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "UUID with all hex chars",
			arg:      "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			wantName: "",
			wantUUID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		},
		{
			name:     "uppercase UUID is treated as name (pattern is lowercase only)",
			arg:      "A1B2C3D4-E5F6-7890-ABCD-EF1234567890",
			wantName: "A1B2C3D4-E5F6-7890-ABCD-EF1234567890",
			wantUUID: "",
		},
		{
			name:     "partial UUID-like string is treated as name",
			arg:      "550e8400-e29b-41d4",
			wantName: "550e8400-e29b-41d4",
			wantUUID: "",
		},
		{
			name:     "empty string is treated as name",
			arg:      "",
			wantName: "",
			wantUUID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, uuid := ResolveInstanceIdentifier(tt.arg)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantUUID, uuid)
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
	expectedFlags := []string{"values", "namespace", "instance-name", "platform", "kubeconfig", "context"}
	for _, name := range expectedFlags {
		flag := cmd.Flags().Lookup(name)
		assert.NotNil(t, flag, "flag %q should be registered", name)
	}
}

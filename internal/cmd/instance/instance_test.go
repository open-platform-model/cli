package instance

import (
	"strings"
	"testing"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
)

// --- 8.1 Unit tests for instance render commands ---

func TestNewInstanceVetCmd(t *testing.T) {
	cmd := NewInstanceVetCmd(&config.GlobalConfig{})
	assert.Equal(t, "vet <instance.cue>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.Args, "should require exactly 1 positional arg")
}

func TestNewInstanceVetCmd_Flags(t *testing.T) {
	cmd := NewInstanceVetCmd(&config.GlobalConfig{})
	assert.NotNil(t, cmd.Flags().Lookup("provider"), "--provider flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("namespace"), "--namespace/-n flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("values"), "--values/-f flag should be registered")
}

func TestNewInstanceBuildCmd(t *testing.T) {
	cmd := NewInstanceBuildCmd(&config.GlobalConfig{})
	assert.Equal(t, "build <instance.cue|module-dir>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewInstanceBuildCmd_Flags(t *testing.T) {
	cmd := NewInstanceBuildCmd(&config.GlobalConfig{})
	assert.NotNil(t, cmd.Flags().Lookup("output"), "--output/-o flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("values"), "--values/-f flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("name"), "--name flag should be registered")
}

func TestNewInstanceApplyCmd(t *testing.T) {
	cmd := NewInstanceApplyCmd(&config.GlobalConfig{})
	assert.Equal(t, "apply <instance.cue>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewInstanceApplyCmd_Flags(t *testing.T) {
	cmd := NewInstanceApplyCmd(&config.GlobalConfig{})
	assert.NotNil(t, cmd.Flags().Lookup("dry-run"), "--dry-run flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("values"), "--values/-f flag should be registered")
}

func TestNewInstanceDiffCmd(t *testing.T) {
	cmd := NewInstanceDiffCmd(&config.GlobalConfig{})
	assert.Equal(t, "diff <instance.cue>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

// --- 8.2 Unit tests for instance cluster-query commands ---

func TestNewInstanceStatusCmd(t *testing.T) {
	cmd := NewInstanceStatusCmd(&config.GlobalConfig{})
	assert.Equal(t, "status <file|name|uuid>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotNil(t, cmd.Args, "should require exactly 1 positional arg")
}

func TestNewInstanceTreeCmd(t *testing.T) {
	cmd := NewInstanceTreeCmd(&config.GlobalConfig{})
	assert.Equal(t, "tree <file|name|uuid>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewInstanceEventsCmd(t *testing.T) {
	cmd := NewInstanceEventsCmd(&config.GlobalConfig{})
	assert.Equal(t, "events <file|name|uuid>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewInstanceStatusCmd_Flags(t *testing.T) {
	cmd := NewInstanceStatusCmd(&config.GlobalConfig{})
	assert.NotNil(t, cmd.Flags().Lookup("namespace"), "--namespace/-n flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("output"), "--output/-o flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("details"), "--details flag should be registered")
}

func TestNewInstanceDeleteCmd(t *testing.T) {
	cmd := NewInstanceDeleteCmd(&config.GlobalConfig{})
	assert.Equal(t, "delete <file|name|uuid>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotNil(t, cmd.Flags().Lookup("dry-run"), "--dry-run flag should be registered")
}

func TestEnsureDeleteAllowed_BlocksControllerManagedInstance(t *testing.T) {
	// inventory.InstanceInventoryRecord / InstanceMetadata are renamed in the X4 slice.
	err := ensureDeleteAllowed(&inventory.InstanceInventoryRecord{CreatedBy: inventory.CreatedByController, InstanceMetadata: inventory.InstanceMetadata{InstanceName: "demo", InstanceNamespace: "apps"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "controller-managed")
}

func TestNewInstanceListCmd(t *testing.T) {
	cmd := NewInstanceListCmd(&config.GlobalConfig{})
	assert.Equal(t, "list", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

// TestInstanceClusterQueryArgParsing tests that positional arg is correctly resolved
// via ResolveInstanceIdentifier inside the command handlers.
// (ResolveInstanceIdentifier lives in cmdutil/flags.go and is renamed in the X4 slice.)
func TestInstanceClusterQueryArgParsing(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		wantName string
		wantUUID string
	}{
		{
			name:     "name identifier",
			arg:      "jellyfin",
			wantName: "jellyfin",
			wantUUID: "",
		},
		{
			name:     "UUID identifier",
			arg:      "550e8400-e29b-41d4-a716-446655440000",
			wantName: "",
			wantUUID: "550e8400-e29b-41d4-a716-446655440000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, uuid := cmdutil.ResolveInstanceIdentifier(tt.arg)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantUUID, uuid)
		})
	}
}

// --- 8.6 Unknown-kind rejection ---
// Was: TestReleaseVetCmd_RejectsBundleRelease (enhancement 0002 D-X3.2). X2 removed
// bundle support; a stray kind: "BundleRelease" file now errors via DetectInstanceKind's
// default ("unknown instance kind"). This test only verifies arg-count enforcement.

func TestInstanceVetCmd_RejectsMissingArg(t *testing.T) {
	cmd := NewInstanceVetCmd(&config.GlobalConfig{})
	cmd.SetArgs([]string{}) // no args

	err := cmd.Execute()
	require.Error(t, err, "vet command should fail without an instance file arg")
}

func TestRunInstanceBuild_RejectsNonManifestOutput(t *testing.T) {
	// cmdutil.InstanceFileFlags is renamed in the X4 slice.
	err := runInstanceBuild("instance.cue", &config.GlobalConfig{}, &cmdutil.InstanceFileFlags{}, "", "", "wide", false, "")
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid output format"))
}

func TestRunInstanceBuild_MissingPath(t *testing.T) {
	err := runInstanceBuild("/nonexistent/instance/path", &config.GlobalConfig{}, &cmdutil.InstanceFileFlags{}, "", "", "yaml", false, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestNewInstanceCmd verifies the instance command group is correctly configured.
func TestNewInstanceCmd(t *testing.T) {
	cmd := NewInstanceCmd(&config.GlobalConfig{})
	assert.Equal(t, "instance", cmd.Use)
	assert.Contains(t, cmd.Aliases, "inst")

	subcommands := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommands[sub.Name()] = true
	}
	for _, expected := range []string{"vet", "build", "apply", "diff", "status", "tree", "events", "delete", "list"} {
		assert.True(t, subcommands[expected], "instance group should have %q subcommand", expected)
	}
}

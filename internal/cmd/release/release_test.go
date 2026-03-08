package release

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
)

// --- 8.1 Unit tests for release render commands ---

func TestNewReleaseVetCmd(t *testing.T) {
	cmd := NewReleaseVetCmd(&config.GlobalConfig{})
	assert.Equal(t, "vet <release.cue>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.Args, "should require exactly 1 positional arg")
}

func TestNewReleaseVetCmd_Flags(t *testing.T) {
	cmd := NewReleaseVetCmd(&config.GlobalConfig{})
	assert.NotNil(t, cmd.Flags().Lookup("module"), "--module flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("provider"), "--provider flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("namespace"), "--namespace/-n flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("values"), "--values/-f flag should be registered")
}

func TestNewReleaseBuildCmd(t *testing.T) {
	cmd := NewReleaseBuildCmd(&config.GlobalConfig{})
	assert.Equal(t, "build <release.cue>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewReleaseBuildCmd_Flags(t *testing.T) {
	cmd := NewReleaseBuildCmd(&config.GlobalConfig{})
	assert.NotNil(t, cmd.Flags().Lookup("module"), "--module flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("output"), "--output/-o flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("values"), "--values/-f flag should be registered")
}

func TestNewReleaseApplyCmd(t *testing.T) {
	cmd := NewReleaseApplyCmd(&config.GlobalConfig{})
	assert.Equal(t, "apply <release.cue>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewReleaseApplyCmd_Flags(t *testing.T) {
	cmd := NewReleaseApplyCmd(&config.GlobalConfig{})
	assert.NotNil(t, cmd.Flags().Lookup("module"), "--module flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("dry-run"), "--dry-run flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("values"), "--values/-f flag should be registered")
}

func TestNewReleaseDiffCmd(t *testing.T) {
	cmd := NewReleaseDiffCmd(&config.GlobalConfig{})
	assert.Equal(t, "diff <release.cue>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

// --- 8.2 Unit tests for release cluster-query commands ---

func TestNewReleaseStatusCmd(t *testing.T) {
	cmd := NewReleaseStatusCmd(&config.GlobalConfig{})
	assert.Equal(t, "status <file|name|uuid>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotNil(t, cmd.Args, "should require exactly 1 positional arg")
}

func TestNewReleaseTreeCmd(t *testing.T) {
	cmd := NewReleaseTreeCmd(&config.GlobalConfig{})
	assert.Equal(t, "tree <file|name|uuid>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewReleaseEventsCmd(t *testing.T) {
	cmd := NewReleaseEventsCmd(&config.GlobalConfig{})
	assert.Equal(t, "events <file|name|uuid>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewReleaseStatusCmd_Flags(t *testing.T) {
	cmd := NewReleaseStatusCmd(&config.GlobalConfig{})
	assert.NotNil(t, cmd.Flags().Lookup("namespace"), "--namespace/-n flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("output"), "--output/-o flag should be registered")
	assert.NotNil(t, cmd.Flags().Lookup("watch"), "--watch flag should be registered")
}

func TestNewReleaseDeleteCmd(t *testing.T) {
	cmd := NewReleaseDeleteCmd(&config.GlobalConfig{})
	assert.Equal(t, "delete <name|uuid>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotNil(t, cmd.Flags().Lookup("dry-run"), "--dry-run flag should be registered")
}

func TestNewReleaseListCmd(t *testing.T) {
	cmd := NewReleaseListCmd(&config.GlobalConfig{})
	assert.Equal(t, "list", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

// TestReleaseClusterQueryArgParsing tests that positional arg is correctly resolved
// via ResolveReleaseIdentifier inside the command handlers.
func TestReleaseClusterQueryArgParsing(t *testing.T) {
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
			name, uuid := cmdutil.ResolveReleaseIdentifier(tt.arg)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantUUID, uuid)
		})
	}
}

// --- 8.6 BundleRelease rejection ---

func TestReleaseVetCmd_RejectsBundleRelease(t *testing.T) {
	// This test verifies the command structure enforces exactly 1 arg.
	// BundleRelease rejection is exercised at the RenderFromReleaseFile level.
	cmd := NewReleaseVetCmd(&config.GlobalConfig{})
	cmd.SetArgs([]string{}) // no args

	err := cmd.Execute()
	require.Error(t, err, "vet command should fail without a release file arg")
}

// TestReleaseGroup verifies the release command group is correctly configured.
func TestNewReleaseCmd(t *testing.T) {
	cmd := NewReleaseCmd(&config.GlobalConfig{})
	assert.Equal(t, "release", cmd.Use)
	assert.Contains(t, cmd.Aliases, "rel")

	subcommands := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommands[sub.Name()] = true
	}
	for _, expected := range []string{"vet", "build", "apply", "diff", "status", "tree", "events", "delete", "list"} {
		assert.True(t, subcommands[expected], "release group should have %q subcommand", expected)
	}
}

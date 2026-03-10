package modulecmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opmodel/cli/internal/config"
)

func TestModTreeCmd_FlagsExist(t *testing.T) {
	cmd := NewModuleTreeCmd(&config.GlobalConfig{})
	assert.Equal(t, "tree", cmd.Use)

	f := cmd.Flags()
	assert.NotNil(t, f.Lookup("namespace"), "--namespace flag should exist")
	assert.NotNil(t, f.Lookup("release-name"), "--release-name flag should exist")
	assert.NotNil(t, f.Lookup("release-id"), "--release-id flag should exist")
	assert.NotNil(t, f.Lookup("depth"), "--depth flag should exist")
	assert.NotNil(t, f.Lookup("output"), "--output/-o flag should exist")
	assert.NotNil(t, f.Lookup("kubeconfig"), "--kubeconfig flag should exist")
	assert.NotNil(t, f.Lookup("context"), "--context flag should exist")
}

func TestModTreeCmd_DefaultFlagValues(t *testing.T) {
	cmd := NewModuleTreeCmd(&config.GlobalConfig{})
	f := cmd.Flags()

	assert.Equal(t, "2", f.Lookup("depth").DefValue, "--depth default should be 2")
	assert.Equal(t, "table", f.Lookup("output").DefValue, "--output default should be table")
}

func TestModTreeCmd_RequiresReleaseSelector(t *testing.T) {
	cmd := NewModuleTreeCmd(&config.GlobalConfig{})
	cmd.SetArgs([]string{"-n", "default"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either --release-name or --release-id is required")
}

func TestModTreeCmd_MutuallyExclusiveSelector(t *testing.T) {
	cmd := NewModuleTreeCmd(&config.GlobalConfig{})
	cmd.SetArgs([]string{"-n", "default", "--release-name", "my-app", "--release-id", "abc123"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--release-name and --release-id are mutually exclusive")
}

func TestModTreeCmd_DepthValidation(t *testing.T) {
	tests := []struct {
		name      string
		depth     string
		wantErr   bool
		errSubstr string
	}{
		{"depth 0 accepted", "0", true, "either --release-name"}, // fails on selector, not depth
		{"depth 1 accepted", "1", true, "either --release-name"},
		{"depth 2 accepted", "2", true, "either --release-name"},
		{"depth -1 rejected", "-1", true, "invalid --depth"},
		{"depth 3 rejected", "3", true, "invalid --depth"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewModuleTreeCmd(&config.GlobalConfig{})
			cmd.SetArgs([]string{"-n", "default", "--depth", tc.depth})
			err := cmd.Execute()
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestModTreeCmd_OutputFormatValidation(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		errSubstr string
	}{
		{"table accepted", "table", "either --release-name"},
		{"json accepted", "json", "either --release-name"},
		{"yaml accepted", "yaml", "either --release-name"},
		{"wide rejected", "wide", "invalid output format"},
		{"dir rejected", "dir", "invalid output format"},
		{"unknown rejected", "xml", "invalid output format"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewModuleTreeCmd(&config.GlobalConfig{})
			cmd.SetArgs([]string{"-n", "default", "-o", tc.format})
			err := cmd.Execute()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errSubstr,
				"format %q: error should contain %q", tc.format, tc.errSubstr)
		})
	}
}

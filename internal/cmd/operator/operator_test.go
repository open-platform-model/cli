package operatorcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/cli/internal/config"
)

func TestNewOperatorCmd(t *testing.T) {
	cmd := NewOperatorCmd(&config.GlobalConfig{})

	assert.Equal(t, "operator", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.ElementsMatch(t, []string{"install", "uninstall"}, names)
}

func TestNewOperatorInstallCmd(t *testing.T) {
	cmd := NewOperatorInstallCmd(&config.GlobalConfig{})

	assert.Equal(t, "install", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	for _, flag := range []string{"crds-only", "rbac", "user", "group", "version", "timeout", "kubeconfig", "context"} {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "expected --%s flag", flag)
	}
}

func TestNewOperatorUninstallCmd(t *testing.T) {
	cmd := NewOperatorUninstallCmd(&config.GlobalConfig{})

	assert.Equal(t, "uninstall", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	for _, flag := range []string{"remove-finalizers", "kubeconfig", "context"} {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "expected --%s flag", flag)
	}
}

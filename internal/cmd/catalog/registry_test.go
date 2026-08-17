package catalogcmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/publish"
)

func TestNewCatalogRegistryCmd(t *testing.T) {
	cmd := NewCatalogRegistryCmd(&config.GlobalConfig{})

	assert.Equal(t, "registry", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "check")
}

func TestNewCatalogRegistryCheckCmd(t *testing.T) {
	cmd := NewCatalogRegistryCheckCmd(&config.GlobalConfig{})

	assert.Equal(t, "check <path@version>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	require.NotNil(t, cmd.Flags().Lookup("compat"))

	// D35's framing is graduation-gated into the help text VERBATIM: the
	// check is an aid, and enforcement exists only at publish.
	assert.Contains(t, cmd.Long, AidSentence)
	assert.Contains(t, cmd.Long, "Exit codes: 0 clean, 2 findings, 3 registry unreachable.")
}

// TestCheckError_ExitCodes pins the check's error → exit-code mapping:
// registry unreachability 3 (the build was never judged), a coordinate no
// build answers to 5, anything else unexpected 1. Findings exit 2 through
// the RunE path, asserted end-to-end.
func TestCheckError_ExitCodes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "connectivity maps to ExitConnectivityError",
			err:  &publish.ConnectivityError{Op: "fetching example.com/catalogs/demo@v1.2.0", Err: errors.New("dial tcp: refused")},
			want: opmexit.ExitConnectivityError,
		},
		{
			name: "not published maps to ExitNotFound",
			err:  fmt.Errorf("example.com/catalogs/demo@v9.9.9: %w", publish.ErrNotPublished),
			want: opmexit.ExitNotFound,
		},
		{
			name: "anything else maps to ExitGeneralError",
			err:  errors.New("evaluating published package failed"),
			want: opmexit.ExitGeneralError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkError(tt.err)
			var exitErr *opmexit.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, tt.want, exitErr.Code)
		})
	}
}

func TestCatalogCmdCarriesRegistryGroup(t *testing.T) {
	cmd := NewCatalogCmd(&config.GlobalConfig{})
	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "registry")
}

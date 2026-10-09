package platformcmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// depPlatform writes a platform module that imports example.com/dep, the
// module the cuemodtest registries serve, pinned at version.
func depPlatform(t *testing.T, version string) string {
	t.Helper()
	return writePlatformModuleDir(t,
		"module: \"example.com/platform@v0\"\nlanguage: version: \"v0.9.0\"\ndeps: \"example.com/dep@v0\": v: \""+version+"\"\n",
		"package platform\n\nimport d \"example.com/dep@v0\"\n\nv: d.version\n")
}

// TestPlatformCheck_BuildFailureExitCodes holds the exit code and the hint
// of a platform that does not build, by cause. A registry that refuses the
// credentials exits 4 with the login hint naming the host; before, it exited
// 2 and printed what the second row prints. A pin the registry does not hold
// keeps exit 2 and its print: the grouped CUE diagnostic, which shows the
// import's position and neither the registry's answer nor the pin hint the
// error carries. That row records what the command prints today, not the
// wanted answer. Every case resolves against a local registry and a cold
// cache.
func TestPlatformCheck_BuildFailureExitCodes(t *testing.T) {
	const (
		login = "Hint: Log in to the registry, then retry:  opm registry login "
		pin   = "Pin a published build in "
	)
	for _, tc := range []struct {
		name     string
		registry func(t *testing.T) string
		version  string
		code     int
		message  string
		shows    func(registry string) string
		notShown string
	}{
		{
			name:     "registry refuses the credentials",
			registry: func(t *testing.T) string { return cuemodtest.StatusRegistry(t, http.StatusUnauthorized) },
			version:  cuemodtest.DepNewest,
			code:     opmexit.ExitPermissionDenied,
			message:  "401 Unauthorized",
			shows:    func(registry string) string { return login + registry },
			notShown: pin,
		},
		{
			name:     "token endpoint refuses the credentials",
			registry: func(t *testing.T) string { return cuemodtest.TokenRegistry(t, http.StatusUnauthorized) },
			version:  cuemodtest.DepNewest,
			code:     opmexit.ExitPermissionDenied,
			message:  "401 Unauthorized",
			shows:    func(registry string) string { return login + registry },
			notShown: pin,
		},
		{
			name:     "registry does not hold the pin",
			registry: cuemodtest.Registry,
			version:  "v0.9.0",
			code:     opmexit.ExitValidationError,
			message:  "import failed",
			shows:    func(string) string { return "platform.cue:3:8" },
			notShown: login,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOCKER_CONFIG", t.TempDir())
			cuemodtest.ColdCache(t)
			registry := tc.registry(t)

			report, diagnostics, err := runCheckAgainst(t, registry, depPlatform(t, tc.version))
			require.Error(t, err)

			var exitErr *opmexit.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, tc.code, exitErr.Code)
			assert.True(t, exitErr.Printed, "the diagnostic is printed once, by the command")
			assert.Contains(t, diagnostics, "platform module does not build")
			assert.Contains(t, diagnostics, tc.message)
			assert.Contains(t, diagnostics, tc.shows(registry))
			assert.NotContains(t, diagnostics, tc.notShown)
			assert.Empty(t, report, "no report is printed for a platform that was never built")
		})
	}
}

// TestNewPlatformCheckCmd_HelpStatesTheRefusedCredentialCode holds the exit
// code a refused registry credential gives in the command's help.
func TestNewPlatformCheckCmd_HelpStatesTheRefusedCredentialCode(t *testing.T) {
	help := NewPlatformCheckCmd(nil).Long
	assert.Contains(t, help, "the command exits 4")
	assert.Contains(t, help, "opm registry login")
}

package instance

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/instinit"
)

// TestAcquireModule_Pinned pins init's exit code for each failure of the
// module acquire: only a registry that gave no response exits 3. A registry
// that answered, whatever it answered, exits 1 today; these rows stay test
// pins rather than spec text.
func TestAcquireModule_Pinned(t *testing.T) {
	held := cuemodtest.Registry
	status := func(code int) func(t *testing.T) string {
		return func(t *testing.T) string { return cuemodtest.StatusRegistry(t, code) }
	}
	token := func(code int) func(t *testing.T) string {
		return func(t *testing.T) string { return cuemodtest.TokenRegistry(t, code) }
	}
	for _, tc := range []struct {
		name     string
		registry func(t *testing.T) string
		module   string
		want     int
	}{
		{"refused connection", func(*testing.T) string { return cuemodtest.UnreachableRegistry }, cuemodtest.DepModule, opmexit.ExitConnectivityError},
		{"module not held", held, "example.com/missing@v0", opmexit.ExitGeneralError},
		{"403", status(http.StatusForbidden), cuemodtest.DepModule, opmexit.ExitGeneralError},
		{"401", status(http.StatusUnauthorized), cuemodtest.DepModule, opmexit.ExitGeneralError},
		{"429", status(http.StatusTooManyRequests), cuemodtest.DepModule, opmexit.ExitGeneralError},
		{"503", status(http.StatusServiceUnavailable), cuemodtest.DepModule, opmexit.ExitGeneralError},
		{"token endpoint answers 401", token(http.StatusUnauthorized), cuemodtest.DepModule, opmexit.ExitGeneralError},
		{"token endpoint answers 403", token(http.StatusForbidden), cuemodtest.DepModule, opmexit.ExitGeneralError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOCKER_CONFIG", t.TempDir())
			cuemodtest.ColdCache(t)
			registry := tc.registry(t)
			mv, err := module.NewVersion(tc.module, cuemodtest.DepNewest)
			require.NoError(t, err)

			_, err = acquireModule(context.Background(), config.NewKernel(registry), mv, registry)
			require.Error(t, err)
			assert.Equal(t, tc.want, initExitCode(t, initError(err)), "%v", err)
		})
	}
}

// TestInitWrite_Pinned pins init's exit code when the staged package's
// dependency closure does not resolve: no response exits 3, a dependency the
// registry does not hold exits 1, and nothing is left behind either way. A
// token endpoint that refuses the caller is a registry that answered, so it
// exits 1 like every other answer, and so does a token endpoint that is rate
// limited (429) or failing (503); while the library read those answers as no
// response, the 401, 429 and 503 rows exited 3 and were printed as "registry
// unreachable". The registry's own answer stays in the message.
func TestInitWrite_Pinned(t *testing.T) {
	token := func(code int) func(t *testing.T) string {
		return func(t *testing.T) string { return cuemodtest.TokenRegistry(t, code) }
	}
	for _, tc := range []struct {
		name     string
		registry func(t *testing.T) string
		imp      string
		want     int
		answer   string
	}{
		{"refused connection", func(*testing.T) string { return cuemodtest.UnreachableRegistry }, cuemodtest.DepModule, opmexit.ExitConnectivityError, ""},
		{"dependency not held", cuemodtest.Registry, "example.com/missing@v0", opmexit.ExitGeneralError, ""},
		{"token endpoint answers 401", token(http.StatusUnauthorized), cuemodtest.DepModule, opmexit.ExitGeneralError, "401 Unauthorized"},
		{"token endpoint answers 403", token(http.StatusForbidden), cuemodtest.DepModule, opmexit.ExitGeneralError, ""},
		{"token endpoint answers 429", token(http.StatusTooManyRequests), cuemodtest.DepModule, opmexit.ExitGeneralError, "429 Too Many Requests"},
		{"token endpoint answers 503", token(http.StatusServiceUnavailable), cuemodtest.DepModule, opmexit.ExitGeneralError, "503 Service Unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOCKER_CONFIG", t.TempDir())
			cuemodtest.ColdCache(t)
			files := instinit.Files{
				instinit.ModuleFile:   []byte("module: \"instance.local/web@v0\"\nlanguage: version: \"v0.9.0\"\n"),
				instinit.InstanceFile: []byte("package instance\n\nimport d \"" + tc.imp + "\"\n\nv: d.version\n"),
				instinit.ValuesFile:   []byte("package instance\n"),
			}
			parent := t.TempDir()
			err := instinit.Write(context.Background(), filepath.Join(parent, "web"), files, tc.registry(t))
			require.Error(t, err)
			assert.Equal(t, tc.want, initExitCode(t, initError(err)), "%v", err)
			if tc.answer != "" {
				assert.Contains(t, err.Error(), tc.answer, "the registry's own answer stays in the message")
				assert.NotContains(t, err.Error(), "unreachable")
			}
			left, readErr := os.ReadDir(parent)
			require.NoError(t, readErr)
			assert.Empty(t, left, "nothing may remain at the target path or beside it")
		})
	}
}

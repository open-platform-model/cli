package instance

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
)

// Every test below runs with the registry mapped to an address that refuses
// connections: a refusal that should come before any registry access exits
// 2, while one that reached the registry would exit 3.

// runInstanceInitCmd runs `opm instance init` in dir with args, feeding stdin
// when non-empty (an injected reader counts as a terminal), and returns the
// error and everything written to standard error.
func runInstanceInitCmd(t *testing.T, dir, stdin string, args ...string) (stderr string, err error) {
	t.Helper()
	wd, wdErr := os.Getwd()
	require.NoError(t, wdErr)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(wd) })

	r, w, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	origStderr := os.Stderr
	os.Stderr = w
	output.SetLogWriter(w)
	defer func() {
		os.Stderr = origStderr
		output.SetLogWriter(origStderr)
	}()

	c := NewInstanceInitCmd(&config.GlobalConfig{Registry: cuemodtest.UnreachableRegistry})
	c.SetArgs(args)
	c.SetOut(new(bytes.Buffer))
	c.SetErr(new(bytes.Buffer))
	if stdin != "" {
		c.SetIn(strings.NewReader(stdin))
	}
	err = c.Execute()

	require.NoError(t, w.Close())
	captured, readErr := io.ReadAll(r)
	require.NoError(t, readErr)
	return string(captured), err
}

func initExitCode(t *testing.T, err error) int {
	t.Helper()
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	return exitErr.Code
}

func TestInstanceInit_Refusals(t *testing.T) {
	for _, tc := range []struct {
		name  string
		stdin string
		args  []string
		want  []string // substrings of the error or standard error
	}{
		{
			name: "module path named twice",
			args: []string{"web", "opmodel.dev/modules/web_app", "--from", "opmodel.dev/modules/web_app", "-n", "demo"},
			want: []string{"named more than once"},
		},
		{
			name: "major suffix",
			args: []string{"web", "opmodel.dev/modules/web_app@v1", "-n", "demo"},
			want: []string{"must not carry a major", "--version v1"},
		},
		{
			name: "major suffix with --from",
			args: []string{"web", "--from", "opmodel.dev/modules/web_app@v1", "-n", "demo"},
			want: []string{"--version v1"},
		},
		{
			name: "missing namespace without a terminal",
			args: []string{"web", "opmodel.dev/modules/web_app"},
			want: []string{"not a terminal", "--namespace"},
		},
		{
			name: "missing module path without a terminal",
			args: []string{"web", "-n", "demo"},
			want: []string{"no module path given", "--from", "<module-path>"},
		},
		{
			name: "missing instance name without a terminal",
			args: []string{"--from", "opmodel.dev/modules/web_app", "-n", "demo"},
			want: []string{"no instance name given"},
		},
		{
			name:  "prompted module path with a major suffix",
			stdin: "opmodel.dev/modules/web_app@v1\n",
			args:  []string{"web", "-n", "demo"},
			want:  []string{"--version v1"},
		},
		{
			name:  "prompted namespace checked like the flag",
			stdin: "Demo\n",
			args:  []string{"web", "opmodel.dev/modules/web_app"},
			want:  []string{`namespace "Demo" is invalid`},
		},
		{
			name:  "empty prompted answer",
			stdin: "\n",
			args:  []string{"opmodel.dev/modules/web_app", "-n", "demo"},
			want:  []string{"instance name must not be empty"},
		},
		{
			name: "invalid instance name",
			args: []string{"Web_App", "opmodel.dev/modules/web_app", "-n", "demo"},
			want: []string{`instance name "Web_App" is invalid`, "lowercase letters"},
		},
		{
			name: "instance name too long",
			args: []string{strings.Repeat("a", 64), "opmodel.dev/modules/web_app", "-n", "demo"},
			want: []string{"at most 63 characters"},
		},
		{
			name: "invalid namespace",
			args: []string{"web", "opmodel.dev/modules/web_app", "-n", "-demo"},
			want: []string{`namespace "-demo" is invalid`},
		},
		{
			name: "invalid version selector",
			args: []string{"web", "opmodel.dev/modules/web_app", "-n", "demo", "--version", "latest"},
			want: []string{"must be vN (float) or X.Y.Z (pin)"},
		},
		{
			name: "invalid package module path",
			args: []string{"web", "opmodel.dev/modules/web_app", "-n", "demo", "--module-path", "Not A Path"},
			want: []string{"--module-path", "is not a module path"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			stderr, err := runInstanceInitCmd(t, dir, tc.stdin, tc.args...)
			require.Error(t, err)
			assert.Equal(t, opmexit.ExitValidationError, initExitCode(t, err), "stderr: %s", stderr)
			for _, w := range tc.want {
				assert.Contains(t, err.Error()+"\n"+stderr, w)
			}
			entries, readErr := os.ReadDir(dir)
			require.NoError(t, readErr)
			assert.Empty(t, entries, "a refusal writes nothing")
		})
	}
}

func TestInstanceInit_ExistingDirectoryRefused(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, dir string)
	}{
		{name: "empty", setup: func(*testing.T, string) {}},
		{name: "holding an instance package", setup: func(t *testing.T, dir string) {
			require.NoError(t, os.WriteFile(filepath.Join(dir, "instance.cue"), []byte("package instance\n"), 0o644))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := t.TempDir()
			target := filepath.Join(parent, "web")
			require.NoError(t, os.Mkdir(target, 0o755))
			tc.setup(t, target)
			before, err := os.ReadDir(target)
			require.NoError(t, err)

			stderr, err := runInstanceInitCmd(t, parent, "", "web", "opmodel.dev/modules/web_app", "-n", "demo")
			require.Error(t, err)
			assert.Equal(t, opmexit.ExitValidationError, initExitCode(t, err))
			assert.Contains(t, err.Error()+stderr, "web already exists")
			assert.Contains(t, stderr, "--dir")
			after, err := os.ReadDir(target)
			require.NoError(t, err)
			assert.Equal(t, before, after, "the existing directory is left as it was")
		})
	}
}

func TestInstanceInit_InsideCUEModuleRefused(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cue.mod", "module.cue"), []byte("module: \"example.com/app@v0\"\n"), 0o644))
	sub := filepath.Join(root, "deploy")
	require.NoError(t, os.Mkdir(sub, 0o755))

	stderr, err := runInstanceInitCmd(t, sub, "", "web", "opmodel.dev/modules/web_app", "-n", "demo")
	require.Error(t, err)
	assert.Equal(t, opmexit.ExitValidationError, initExitCode(t, err))
	resolvedRoot, evalErr := filepath.EvalSymlinks(root)
	require.NoError(t, evalErr)
	msg := err.Error() + stderr
	assert.True(t, strings.Contains(msg, root) || strings.Contains(msg, resolvedRoot), "names the enclosing module root: %s", msg)
	assert.Contains(t, msg, "inside the CUE module")
	assert.Contains(t, stderr, "outside")
	_, statErr := os.Stat(filepath.Join(sub, "web"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestInstanceInit_RegistryUnreachable(t *testing.T) {
	cuemodtest.ColdCache(t)
	dir := t.TempDir()
	stderr, err := runInstanceInitCmd(t, dir, "", "web", "opmodel.dev/modules/web_app", "-n", "demo")
	require.Error(t, err)
	assert.Equal(t, opmexit.ExitConnectivityError, initExitCode(t, err), "stderr: %s", stderr)
	assert.Contains(t, err.Error(), "opmodel.dev/modules/web_app")
	assert.Contains(t, err.Error(), "127.0.0.1:1")
	entries, readErr := os.ReadDir(dir)
	require.NoError(t, readErr)
	assert.Empty(t, entries)
}

func TestCollectInputs(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		flags initFlags
		want  initInputs
	}{
		{
			name:  "positional module path",
			args:  []string{"web", "opmodel.dev/modules/web_app"},
			flags: initFlags{namespace: "demo"},
			want: initInputs{
				name: "web", namespace: "demo", dir: "web",
				packageModulePath: "instance.local/web@v0", path: "opmodel.dev/modules/web_app",
			},
		},
		{
			name:  "from flag, version, dir and module path",
			args:  []string{"web"},
			flags: initFlags{from: "opmodel.dev/modules/web_app", version: "v1", namespace: "demo", dir: "deploy/web", modulePath: "example.com/deploy/web@v0"},
			want: initInputs{
				name: "web", namespace: "demo", dir: "deploy/web",
				packageModulePath: "example.com/deploy/web@v0", path: "opmodel.dev/modules/web_app",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewInstanceInitCmd(&config.GlobalConfig{})
			got, err := collectInputs(c, tc.args, tc.flags)
			require.NoError(t, err)
			tc.want.selector = got.selector
			assert.Equal(t, tc.want, *got)
		})
	}
}

func TestCollectInputs_PromptsInOrder(t *testing.T) {
	c := NewInstanceInitCmd(&config.GlobalConfig{})
	c.SetIn(strings.NewReader("web\nopmodel.dev/modules/web_app\ndemo\n"))
	got, err := collectInputs(c, nil, initFlags{})
	require.NoError(t, err)
	assert.Equal(t, "web", got.name)
	assert.Equal(t, "opmodel.dev/modules/web_app", got.path)
	assert.Equal(t, "demo", got.namespace)
}

func TestClassifyInitArgs(t *testing.T) {
	for _, tc := range []struct {
		args       []string
		name, path string
	}{
		{args: nil},
		{args: []string{"web"}, name: "web"},
		{args: []string{"opmodel.dev/modules/web_app"}, path: "opmodel.dev/modules/web_app"},
		{args: []string{"web", "opmodel.dev/modules/web_app"}, name: "web", path: "opmodel.dev/modules/web_app"},
	} {
		name, path := classifyInitArgs(tc.args)
		assert.Equal(t, tc.name, name, "args %v", tc.args)
		assert.Equal(t, tc.path, path, "args %v", tc.args)
	}
}

func TestNewInstanceInitCmd_Flags(t *testing.T) {
	cmd := NewInstanceInitCmd(&config.GlobalConfig{})
	assert.Equal(t, "init [instance-name] [module-path]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Long, "opmodel.dev/modules/cert_manager")
	assert.Contains(t, cmd.Long, "opmodel.dev/modules/web_app")

	for _, f := range []struct{ name, shorthand, def string }{
		{name: "from"},
		{name: "version"},
		{name: "namespace", shorthand: "n"},
		{name: "dir"},
		{name: "module-path"},
	} {
		flag := cmd.Flags().Lookup(f.name)
		require.NotNil(t, flag, "--%s", f.name)
		assert.Equal(t, f.shorthand, flag.Shorthand, "--%s shorthand", f.name)
		assert.Equal(t, f.def, flag.DefValue, "--%s default", f.name)
	}
}

func TestInstanceHelp_ListsInit(t *testing.T) {
	cmd := NewInstanceCmd(&config.GlobalConfig{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	require.NoError(t, cmd.Execute())
	assert.Regexp(t, `(?m)^\s+init\s+Create an instance package`, out.String())
}

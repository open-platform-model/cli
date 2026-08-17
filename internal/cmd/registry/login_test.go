package registrycmd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelabs.dev/go/oci/ociregistry/ocimem"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/dockercfg"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/publish"
)

func TestNewRegistryCmd(t *testing.T) {
	cmd := NewRegistryCmd(&config.GlobalConfig{})

	assert.Equal(t, "registry", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	assert.Contains(t, names, "login")
}

func TestNewRegistryLoginCmd(t *testing.T) {
	cmd := NewRegistryLoginCmd(&config.GlobalConfig{})

	assert.Equal(t, "login [host]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Long, "Exit codes: 0 written, 2 refusal, 3 registry unreachable.")
	assert.Contains(t, cmd.Long, "docker login")
}

func TestParseHostArg(t *testing.T) {
	tests := []struct {
		arg  string
		want target
	}{
		{arg: "ghcr.io", want: target{host: "ghcr.io"}},
		{arg: "localhost:5000+insecure", want: target{host: "localhost:5000", insecure: true}},
		{arg: "registry.example.com:8443", want: target{host: "registry.example.com:8443"}},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			got := parseHostArg(tt.arg)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.arg, got.arg())
		})
	}
}

func TestResolveTarget_SingleHost(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		want     target
	}{
		{
			// A prefix mapping alone gains CUE's default fallback registry
			// as a second host; single-host means the fallback is the same
			// host too.
			name:     "secure host",
			registry: "opmodel.dev=ghcr.io/open-platform-model,ghcr.io/fallback",
			want:     target{host: "ghcr.io"},
		},
		{
			name:     "insecure host keeps its scheme choice",
			registry: "testing.opmodel.dev=localhost:5000+insecure,localhost:5000+insecure",
			want:     target{host: "localhost:5000", insecure: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgt, refusal := resolveTarget(tt.registry)
			require.Nil(t, refusal)
			assert.Equal(t, tt.want, tgt)
		})
	}
}

// TestResolveTarget_MultiHostRefuses pins the refusal against the shipped
// default mapping: every host listed, each action a runnable command.
func TestResolveTarget_MultiHostRefuses(t *testing.T) {
	_, refusal := resolveTarget(config.DefaultRegistry)
	require.NotNil(t, refusal)

	assert.Contains(t, refusal.Headline, "name the one to log in to")
	hosts := make([]string, 0, len(refusal.Evidence))
	for _, row := range refusal.Evidence {
		hosts = append(hosts, row[0])
	}
	assert.Contains(t, hosts, "ghcr.io")
	assert.Contains(t, hosts, "registry.cue.works")
	assert.Contains(t, refusal.Action, "opm registry login ghcr.io")
	assert.Contains(t, refusal.Action, "opm registry login registry.cue.works")
}

func TestResolveTarget_NoRegistryRefuses(t *testing.T) {
	_, refusal := resolveTarget("")
	require.NotNil(t, refusal)
	assert.Contains(t, refusal.Headline, "no registry is configured")
	assert.Contains(t, refusal.Action, "opm config init")
}

func TestResolveTarget_MalformedMappingRefuses(t *testing.T) {
	_, refusal := resolveTarget("opmodel.dev=not a host!,=,=")
	require.NotNil(t, refusal)
	assert.Contains(t, refusal.Headline, "does not parse")
	assert.Error(t, refusal.Err)
}

// captureStderr runs fn with os.Stderr redirected to a pipe and returns
// what was written — the refusal detail block prints there directly.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() { os.Stderr = old }()
	fn()
	require.NoError(t, w.Close())
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(data)
}

// TestLoginCmd_NonTTYRefuses: go test runs without a terminal on stdin,
// which is exactly the state the command must refuse in — pointing at
// docker login, the scripted path that writes the same file.
func TestLoginCmd_NonTTYRefuses(t *testing.T) {
	t.Setenv("DOCKER_AUTH_CONFIG", "")
	var buf bytes.Buffer
	output.SetLogWriter(&buf)

	cmd := NewRegistryLoginCmd(&config.GlobalConfig{})
	cmd.SetArgs([]string{"registry.example.com"})
	var err error
	stderr := captureStderr(t, func() { err = cmd.Execute() })

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.True(t, exitErr.Printed)
	assert.Contains(t, err.Error(), "standard input is not a terminal")
	assert.Contains(t, stderr, "docker login registry.example.com")
}

// TestLoginCmd_DockerAuthConfigNotice: the env var shadows the file on the
// read side, so a login that writes the file says so.
func TestLoginCmd_DockerAuthConfigNotice(t *testing.T) {
	t.Setenv("DOCKER_AUTH_CONFIG", `{"auths":{}}`)
	var buf bytes.Buffer
	output.SetLogWriter(&buf)

	cmd := NewRegistryLoginCmd(&config.GlobalConfig{})
	cmd.SetArgs([]string{"registry.example.com"})
	err := cmd.Execute()

	require.Error(t, err) // still the non-TTY refusal downstream
	assert.Contains(t, buf.String(), "DOCKER_AUTH_CONFIG is set")
}

// TestLoginCmd_MultiHostRefusalEndToEnd drives the no-arg form against the
// shipped default mapping through the command surface.
func TestLoginCmd_MultiHostRefusalEndToEnd(t *testing.T) {
	t.Setenv("DOCKER_AUTH_CONFIG", "")
	var buf bytes.Buffer
	output.SetLogWriter(&buf)

	cfg := &config.GlobalConfig{Registry: config.DefaultRegistry}
	cmd := NewRegistryLoginCmd(cfg)
	cmd.SetArgs([]string{})
	var err error
	stderr := captureStderr(t, func() { err = cmd.Execute() })

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.Contains(t, buf.String(), "maps to 2 hosts")
	assert.Contains(t, stderr, "opm registry login ghcr.io")
}

// basicAuthServer is a minimal /v2/ endpoint guarding with basic auth.
func basicAuthServer(t *testing.T, user, pass string) target {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	host := strings.TrimPrefix(srv.URL, "http://")
	return target{host: host, insecure: true}
}

func TestProbeCredential(t *testing.T) {
	tgt := basicAuthServer(t, "opm", "sesame")

	t.Run("accepted credential passes", func(t *testing.T) {
		require.NoError(t, probeCredential(context.Background(), tgt, "opm", "sesame"))
	})

	t.Run("rejected credential is errBadCredential", func(t *testing.T) {
		err := probeCredential(context.Background(), tgt, "opm", "wrong")
		require.ErrorIs(t, err, errBadCredential)
	})

	t.Run("forbidden is errBadCredential", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		t.Cleanup(srv.Close)
		err := probeCredential(context.Background(), target{host: strings.TrimPrefix(srv.URL, "http://"), insecure: true}, "u", "s")
		require.ErrorIs(t, err, errBadCredential)
	})

	t.Run("unreachable host is connectivity", func(t *testing.T) {
		err := probeCredential(context.Background(), target{host: "127.0.0.1:1", insecure: true}, "u", "s")
		var connErr *publish.ConnectivityError
		require.ErrorAs(t, err, &connErr)
	})
}

// TestLogin_BadCredentialLeavesFileUntouched: verify-then-write means a
// rejected credential never reaches the file.
func TestLogin_BadCredentialLeavesFileUntouched(t *testing.T) {
	tgt := basicAuthServer(t, "opm", "sesame")
	path := filepath.Join(t.TempDir(), "config.json")

	err := login(context.Background(), path, tgt, "opm", "wrong")
	require.ErrorIs(t, err, errBadCredential)
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr))
}

// identitySchemaStub mirrors publish's hermetic test schema — just enough
// of core's #IdentityPackage for the pipeline to unify against.
const identitySchemaStub = `
#IdentityPackage: {
	ModulePath!: string & =~"^[a-z0-9._-]+(/[a-z0-9._-]+)*@v[0-9]+$"
	Version!:    string & =~"^\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
}
`

// TestLogin_InversionAgainstRegistry is the hermetic inversion of publish's
// TestPush_Authenticated: there, a hand-written docker config makes the
// authenticated push succeed; here, `login` itself writes the file against
// the in-process registry's basic auth, and the pipeline's push succeeds
// using only what it wrote.
func TestLogin_InversionAgainstRegistry(t *testing.T) {
	reg, err := modregistrytest.NewServer(
		ocimem.NewWithConfig(&ocimem.Config{ImmutableTags: true}),
		&modregistrytest.AuthConfig{Username: "opm", Password: "sesame"},
	)
	require.NoError(t, err)
	t.Cleanup(reg.Close)

	dockerDir := t.TempDir()
	t.Setenv("DOCKER_CONFIG", dockerDir)
	path, err := dockercfg.Path()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dockerDir, "config.json"), path)

	// The command's verify-then-write core, exactly as runLogin calls it.
	tgt := target{host: reg.Host(), insecure: true}
	require.NoError(t, login(context.Background(), path, tgt, "opm", "sesame"))

	// The pipeline's push, authenticating with only the written file.
	dir := t.TempDir()
	files := map[string]string{
		"cue.mod/module.cue":    "module: \"example.com/modules/demo@v1\"\nlanguage: version: \"v0.17.0\"\nsource: kind: \"self\"\n",
		"identity/identity.cue": "package identity\n\nModulePath: \"example.com/modules/demo@v1\"\nVersion:    \"1.2.0\"\n",
		"module.cue":            "package demo\n\nmetadata: {\n\tname:       \"demo\"\n\tmodulePath: \"example.com/modules/demo@v1\"\n\tversion:    \"1.2.0\"\n}\n",
	}
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}

	cueCtx := cuecontext.New()
	schema := cueCtx.CompileString(identitySchemaStub)
	require.NoError(t, schema.Err())
	opts := publish.Options{
		Dir:            dir,
		Kind:           publish.KindModule,
		Context:        cueCtx,
		IdentitySchema: schema.LookupPath(cue.MakePath(cue.Def("IdentityPackage"))),
		Registry:       reg.Host() + "+insecure",
	}
	plan, err := publish.Run(context.Background(), opts)
	require.NoError(t, err)
	require.True(t, plan.Go())
	require.NoError(t, publish.Push(context.Background(), opts, plan))
}

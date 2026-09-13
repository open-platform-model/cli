package cmdutil

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
)

// writeInstancePackage writes an import-free CUE module holding instance.cue
// with body into a temp directory and returns the directory. Import-free, so
// the kernel's acquire needs no registry to reach its shape gate.
func writeInstancePackage(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"),
		[]byte("module: \"test.example/instance@v0\"\nlanguage: version: \"v0.17.0\"\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "instance.cue"), []byte(body), 0o600))
	return dir
}

// skipIfRegistryUnavailable skips when err looks like the registry could not
// be reached (the repo's posture for registry-backed unit tests).
func skipIfRegistryUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := err.Error()
	for _, needle := range []string{"dial tcp", "no such host", "connection refused", "i/o timeout", "context deadline exceeded", "TLS handshake", "network is unreachable"} {
		if strings.Contains(msg, needle) {
			t.Skipf("registry unavailable: %v", err)
		}
	}
}

func TestIsInstancePath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "instance.cue")
	require.NoError(t, os.WriteFile(file, []byte("package test\n"), 0o600))

	tests := []struct {
		name string
		arg  string
		want bool
	}{
		{name: "existing directory", arg: dir, want: true},
		{name: "existing file", arg: file, want: true},
		{name: "missing file with .cue suffix", arg: "missing.cue", want: true},
		{name: "relative dot path", arg: "./jellyfin", want: true},
		{name: "home path", arg: "~/jellyfin", want: true},
		{name: "path with separator", arg: filepath.Join("apps", "jellyfin"), want: true},
		{name: "instance name", arg: "jellyfin", want: false},
		{name: "instance name with hyphens", arg: "my-app-prod", want: false},
		{name: "uuid", arg: "550e8400-e29b-41d4-a716-446655440000", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isInstancePath(tt.arg))
		})
	}
}

func TestResolveInstanceArg_NameAndUUIDForms(t *testing.T) {
	ctx := context.Background()
	cfg := &config.GlobalConfig{}

	got, err := ResolveInstanceArg(ctx, "jellyfin", cfg)
	require.NoError(t, err)
	assert.Equal(t, InstanceArg{Name: "jellyfin"}, got)

	got, err = ResolveInstanceArg(ctx, "550e8400-e29b-41d4-a716-446655440000", cfg)
	require.NoError(t, err)
	assert.Equal(t, InstanceArg{UUID: "550e8400-e29b-41d4-a716-446655440000"}, got)
}

// TestResolveInstanceArg_ImportFreeInstancePackage acquires an instance
// package that carries only what the kernel's shape gate needs (kind,
// concrete identity, an embedded #module of kind Module) and so resolves
// without a registry; the refusals are the gate's, framed with the argument
// the user passed. The valid case is the shape of the inst-tree integration
// fixture (tests/integration/inst-tree/testdata).
func TestResolveInstanceArg_ImportFreeInstancePackage(t *testing.T) {
	const module = `#module: {
	kind: "Module"
	metadata: {
		name:       "jellyfin"
		modulePath: "test.example/modules/jellyfin@v0"
		version:    "0.0.1"
	}
}
`
	tests := []struct {
		name    string
		body    string
		want    InstanceArg
		wantErr string
	}{
		{
			name: "valid instance resolves to its declared name and namespace",
			body: "package jellyfin\n\nkind: \"ModuleInstance\"\nmetadata: {\n\tname:      \"jellyfin\"\n\tnamespace: \"media\"\n}\n" + module,
			want: InstanceArg{Name: "jellyfin", Namespace: "media"},
		},
		{
			name:    "kind that is not ModuleInstance is refused",
			body:    "package jellyfin\n\nkind: \"Module\"\nmetadata: {\n\tname:      \"jellyfin\"\n\tnamespace: \"media\"\n}\n" + module,
			wantErr: `expected kind "ModuleInstance"`,
		},
		{
			name:    "open namespace is refused",
			body:    "package jellyfin\n\nkind: \"ModuleInstance\"\nmetadata: {\n\tname:      \"jellyfin\"\n\tnamespace: string\n}\n" + module,
			wantErr: `"metadata.namespace" is not concrete`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeInstancePackage(t, tt.body)
			for _, arg := range []string{dir, filepath.Join(dir, "instance.cue")} {
				got, err := ResolveInstanceArg(context.Background(), arg, &config.GlobalConfig{})
				if tt.wantErr != "" {
					require.Error(t, err, "arg %q", arg)
					assert.Contains(t, err.Error(), arg, "the error names the path")
					assert.Contains(t, err.Error(), tt.wantErr, "arg %q", arg)
					continue
				}
				require.NoError(t, err, "arg %q", arg)
				assert.Equal(t, tt.want, got, "arg %q", arg)
			}
		})
	}
}

// TestResolveInstanceArg_RegistryBackedInstancePackage acquires the
// operator-owned e2e fixture, a renderable instance package that imports
// core and the podinfo fixture module from GHCR, through the kernel with the
// shipped default registry, and skips when the registry is unreachable.
func TestResolveInstanceArg_RegistryBackedInstancePackage(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "tests", "e2e", "testdata", "operator-owned"))
	require.NoError(t, err)
	if _, statErr := os.Stat(filepath.Join(dir, "instance.cue")); statErr != nil {
		t.Skipf("fixture not found: %v", statErr)
	}

	cfg := &config.GlobalConfig{Registry: config.DefaultRegistry}
	for _, arg := range []string{dir, filepath.Join(dir, "instance.cue")} {
		got, err := ResolveInstanceArg(context.Background(), arg, cfg)
		skipIfRegistryUnavailable(t, err)
		require.NoError(t, err, "arg %q", arg)
		assert.Equal(t, InstanceArg{Name: "e2e-operator-owned", Namespace: "default"}, got, "arg %q", arg)
		assert.Equal(t, "default", got.ToSelectorFlags("").Namespace, "the file's namespace applies without --namespace")
		assert.Equal(t, "staging", got.ToSelectorFlags("staging").Namespace, "--namespace wins over the file")
	}
}

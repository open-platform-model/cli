package dockercfg

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPath(t *testing.T) {
	t.Run("DOCKER_CONFIG wins", func(t *testing.T) {
		t.Setenv("DOCKER_CONFIG", filepath.Join("some", "dir"))
		p, err := Path()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join("some", "dir", "config.json"), p)
	})

	t.Run("falls back to ~/.docker", func(t *testing.T) {
		t.Setenv("DOCKER_CONFIG", "")
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		p, err := Path()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, ".docker", "config.json"), p)
	})
}

// decode parses a written config file back into its raw envelope and auths.
func decode(t *testing.T, path string) (top, auths map[string]json.RawMessage) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &top))
	require.NoError(t, json.Unmarshal(top["auths"], &auths))
	return top, auths
}

func TestUpsert_FreshFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".docker", "config.json")

	require.NoError(t, Upsert(path, "ghcr.io", "user", "sesame"))

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}

	_, auths := decode(t, path)
	var entry struct {
		Auth string `json:"auth"`
	}
	require.NoError(t, json.Unmarshal(auths["ghcr.io"], &entry))
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("user:sesame")), entry.Auth)

	// Tab-indented, trailing newline — the shape docker itself writes.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "\n\t\"auths\"")
	assert.True(t, len(data) > 0 && data[len(data)-1] == '\n')
}

// compact returns s with insignificant JSON whitespace removed. The writer
// re-encodes the whole file with tab indentation (by design), so foreign
// values are preserved content-exact: byte-identical after compaction, and
// literally byte-identical once the file is in the writer's canonical form
// (TestUpsert_Idempotent pins that fixed point).
func compact(t *testing.T, s string) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, json.Compact(&buf, []byte(s)))
	return buf.String()
}

// TestUpsert_SharedFilePreserved pins the writer's whole reason to exist:
// everything outside the edited entry — credsStore, credHelpers, foreign
// hosts, keys this package has never heard of — survives the rewrite intact.
func TestUpsert_SharedFilePreserved(t *testing.T) {
	fixture := map[string]string{
		"credsStore":  `"desktop"`,
		"credHelpers": `{"gcr.io":"gcloud"}`,
		"plugins":     `{"buildx":{"enabled":"true"}}`,
		"futureKey":   `[1,2,{"deep":true}]`,
	}
	foreignAuth := `{"auth":"Zm9vOmJhcg==","email":"foo@example.com"}`

	path := filepath.Join(t.TempDir(), "config.json")
	seed := `{
	"auths": {
		"registry.example.com": ` + foreignAuth + `
	},
	"credsStore": ` + fixture["credsStore"] + `,
	"credHelpers": ` + fixture["credHelpers"] + `,
	"plugins": ` + fixture["plugins"] + `,
	"futureKey": ` + fixture["futureKey"] + `
}
`
	require.NoError(t, os.WriteFile(path, []byte(seed), 0o644))

	require.NoError(t, Upsert(path, "ghcr.io", "user", "sesame"))

	top, auths := decode(t, path)
	for key, want := range fixture {
		assert.Equal(t, compact(t, want), compact(t, string(top[key])), key)
	}
	assert.Equal(t, compact(t, foreignAuth), compact(t, string(auths["registry.example.com"])))

	var entry struct {
		Auth string `json:"auth"`
	}
	require.NoError(t, json.Unmarshal(auths["ghcr.io"], &entry))
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("user:sesame")), entry.Auth)

	// An existing file keeps its own permissions — only creation is 0600.
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	}
}

// TestUpsert_Idempotent: the second identical write produces identical
// bytes — the writer's encoding is a fixed point of itself.
func TestUpsert_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, Upsert(path, "ghcr.io", "user", "sesame"))
	first, err := os.ReadFile(path)
	require.NoError(t, err)

	require.NoError(t, Upsert(path, "ghcr.io", "user", "sesame"))
	second, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second))
}

// TestUpsert_ReplacesOnlyTargetHost: re-login with a new secret replaces the
// one entry and nothing else.
func TestUpsert_ReplacesOnlyTargetHost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, Upsert(path, "ghcr.io", "user", "old"))
	require.NoError(t, Upsert(path, "localhost:5000", "dev", "devpass"))
	require.NoError(t, Upsert(path, "ghcr.io", "user", "new"))

	_, auths := decode(t, path)
	require.Len(t, auths, 2)
	var entry struct {
		Auth string `json:"auth"`
	}
	require.NoError(t, json.Unmarshal(auths["ghcr.io"], &entry))
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("user:new")), entry.Auth)
	require.NoError(t, json.Unmarshal(auths["localhost:5000"], &entry))
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("dev:devpass")), entry.Auth)
}

func TestUpsert_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "malformed JSON refused",
			content: `{"auths": {`,
			wantErr: "not valid JSON",
		},
		{
			name:    "non-object auths refused",
			content: `{"auths": "surprise"}`,
			wantErr: "auths is not a JSON object",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o600))

			err := Upsert(path, "ghcr.io", "user", "sesame")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)

			// Refused means untouched.
			data, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			assert.Equal(t, tt.content, string(data))
		})
	}
}

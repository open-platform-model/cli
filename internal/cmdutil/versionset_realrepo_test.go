package cmdutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/publish"
)

// TestRunVersionSet_RealRepoFixtures is enhancement 0011's graduation gate
// for version set: the fixtures are byte-for-byte copies of the shipped
// identity files (catalog_opm/src/identity/identity.cue and
// modules/web_app/identity/identity.cue). On both, setting the declared
// version is a no-op (identical bytes, untouched mtime), and setting a new
// version changes exactly the version literal — the defaulted-disjunction
// assertion and the release-automation marker line survive.
func TestRunVersionSet_RealRepoFixtures(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		kind     publish.Kind
		declared string
		next     string
		// marker is a line fragment that must survive the write verbatim.
		marker string
	}{
		{
			name:     "catalog_opm identity",
			fixture:  "catalog_opm_identity.cue",
			kind:     publish.KindCatalog,
			declared: "2.0.0-alpha.3",
			next:     "2.0.0-alpha.4",
			marker:   `Version: #VersionType | *"2.0.0-alpha.4" // x-release-please-version`,
		},
		{
			name:     "web_app identity",
			fixture:  "web_app_identity.cue",
			kind:     publish.KindModule,
			declared: "1.0.0",
			next:     "1.1.0",
			marker:   `Version: #VersionType | *"1.1.0"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original, err := os.ReadFile(filepath.Join("testdata", "versionset", tt.fixture))
			require.NoError(t, err)

			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "identity"), 0o755))
			path := filepath.Join(dir, "identity", "identity.cue")
			require.NoError(t, os.WriteFile(path, original, 0o644))

			// Idempotency: the declared version is a no-op — identical bytes,
			// untouched mtime.
			past := time.Now().Add(-time.Hour)
			require.NoError(t, os.Chtimes(path, past, past))
			before, err := os.Stat(path)
			require.NoError(t, err)

			require.NoError(t, RunVersionSet(tt.kind, tt.declared, []string{dir}))

			data, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, string(original), string(data), "no-op must not rewrite any byte")
			after, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, before.ModTime(), after.ModTime(), "no-op must not touch the file")

			// A real set changes exactly the version literal: the whole file
			// equals the original with one quoted value swapped, so comments,
			// the defaulted-disjunction assertion, and the marker line shape
			// all survive byte-for-byte.
			require.NoError(t, RunVersionSet(tt.kind, tt.next, []string{dir}))

			data, err = os.ReadFile(path)
			require.NoError(t, err)
			want := strings.Replace(string(original),
				`*"`+tt.declared+`"`, `*"`+tt.next+`"`, 1)
			require.NotEqual(t, string(original), want, "fixture must carry the declared default")
			assert.Equal(t, want, string(data))
			assert.Contains(t, string(data), tt.marker)
		})
	}
}

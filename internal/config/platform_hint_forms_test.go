package config

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelabs.dev/go/oci/ociregistry/ocimem"
	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// The four hints a platform build failure can get beside the shape hint.
const (
	hintLogin   = "Log in to the registry, then retry:  opm registry login"
	hintPin     = "Pin a published build in "
	hintDefault = "Fix the platform module at "
	hintKey     = "Each #registry entry's key must equal the module path of the catalog it imports (#catalog)"
)

// brokenModFileRegistry serves example.com/dep v0.2.0 with a module file
// that does not parse. modregistrytest refuses to publish such a module, so
// the manifest and its two layers are pushed raw.
func brokenModFileRegistry(t *testing.T) string {
	t.Helper()
	const (
		repo       = "example.com/dep"
		manifestMT = "application/vnd.oci.image.manifest.v1+json"
		modFile    = "module: \"example.com/dep@v0\"\nlanguage: version: \"v0.9.0\"\nbogus: 1\n"
	)
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	for name, data := range map[string]string{"cue.mod/module.cue": modFile, "dep.cue": "package dep\n\nversion: \"v0.2.0\"\n"} {
		f, err := zw.Create(name)
		require.NoError(t, err)
		_, err = f.Write([]byte(data))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())

	ctx := context.Background()
	reg := ocimem.New()
	push := func(mediaType string, b []byte) ociregistry.Descriptor {
		sum := sha256.Sum256(b)
		d := ociregistry.Descriptor{MediaType: mediaType, Size: int64(len(b)), Digest: ociregistry.Digest("sha256:" + hex.EncodeToString(sum[:]))}
		_, err := reg.PushBlob(ctx, repo, d, bytes.NewReader(b))
		require.NoError(t, err)
		return d
	}
	manifest, err := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     manifestMT,
		"config":        push("application/vnd.cue.module.v1+json", []byte("{}")),
		"layers": []any{
			push("application/zip", archive.Bytes()),
			push("application/vnd.cue.modulefile.v1", []byte(modFile)),
		},
	})
	require.NoError(t, err)
	_, err = reg.PushManifest(ctx, repo, cuemodtest.DepNewest, manifest, manifestMT)
	require.NoError(t, err)

	srv, err := modregistrytest.NewServer(reg, nil)
	require.NoError(t, err)
	t.Cleanup(srv.Close)
	return srv.Host() + "+insecure"
}

// nestedDepRegistry serves example.com/platform/dep, a module whose path
// lies inside the platform module's own path, so a dep/ directory in the
// platform module provides the same package.
func nestedDepRegistry(t *testing.T) string {
	t.Helper()
	reg, err := modregistrytest.New(fstest.MapFS{
		"example.com_platform_dep_v0.0.1/cue.mod/module.cue": &fstest.MapFile{Data: []byte("module: \"example.com/platform/dep@v0\"\nlanguage: version: \"v0.9.0\"\n")},
		"example.com_platform_dep_v0.0.1/dep.cue":            &fstest.MapFile{Data: []byte("package dep\n\nversion: \"v0.0.1\"\n")},
	}, "")
	require.NoError(t, err)
	t.Cleanup(reg.Close)
	return reg.Host() + "+insecure"
}

// TestPlatformBuildHint_Forms pins the hint, and the validation exit class,
// of the platform build failures TestPlatformBuildHint_Pinned does not
// drive. Every case resolves against a local registry and a cold cache. Two
// rows changed their hint when the hint moved from the message text to the
// error type; each states the old and the new text. The exit class did not
// move. The refused credential has its own test,
// TestPlatformBuildHint_RefusedCredential.
func TestPlatformBuildHint_Forms(t *testing.T) {
	for _, tc := range []struct {
		name     string
		registry func(t *testing.T) string
		deps     string
		src      string
		extra    map[string]string
		contains string
		want     string
	}{
		{
			name:     "directly imported dependency whose module file does not parse",
			registry: brokenModFileRegistry,
			deps:     depPinned,
			src:      importsDep,
			contains: "bogus: field not allowed",
			// Old hint, from the message text, which carried neither matched
			// phrase: "Fix the platform module at <dir> (pins in
			// <dir>/cue.mod/module.cue), then try again". New hint, from the
			// library's resolution kind, the one this defect already got when
			// it was met while the module graph is expanded: "Pin a published
			// build in <dir>/cue.mod/module.cue, then try again".
			want: hintPin,
		},
		{
			name:     "import whose package name does not match",
			registry: cuemodtest.Registry,
			deps:     depPinned,
			src:      "package platform\n\nimport d \"example.com/dep@v0:other\"\n\nv: d.version\n",
			contains: "no files in package directory with package name",
			// Old hint, from cue/load's "cannot find package" prefix: "Pin a
			// published build in <dir>/cue.mod/module.cue, then try again".
			// New hint, because this is neither a fetch nor a resolution
			// failure and no other pin cures it: "Fix the platform module at
			// <dir> (pins in <dir>/cue.mod/module.cue), then try again".
			want: hintDefault,
		},
		{
			name:     "ambiguous import",
			registry: nestedDepRegistry,
			deps:     "deps: \"example.com/platform/dep@v0\": v: \"v0.0.1\"\n",
			src:      "package platform\n\nimport d \"example.com/platform/dep\"\n\nv: d.version\n",
			extra:    map[string]string{"dep/dep.cue": "package dep\n\nversion: \"local\"\n"},
			contains: "ambiguous import",
			want:     hintPin,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cuemodtest.ColdCache(t)
			dir := hintPlatform(t, tc.deps, tc.src)
			for name, data := range tc.extra {
				path := filepath.Join(dir, filepath.FromSlash(name))
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
				require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
			}
			_, err := BuildPlatformModule(context.Background(), dir, tc.registry(t))
			require.Error(t, err)
			require.ErrorIs(t, err, oerrors.ErrValidation)
			var detail *oerrors.DetailError
			require.True(t, errors.As(err, &detail), "%v", err)
			assert.Contains(t, detail.Message, tc.contains)
			assert.Contains(t, detail.Hint, tc.want, "%s", detail.Message)
		})
	}
}

// blobAnswers serves the fixture registry with every blob of example.com/dep
// answering status: the tag lookup passes and the archive fetch is refused.
func blobAnswers(status int) func(t *testing.T) string {
	return func(t *testing.T) string {
		return cuemodtest.Fronted(t, cuemodtest.Registry(t), cuemodtest.BlobAnswers("example.com/dep", status))
	}
}

// TestPlatformBuildHint_RefusedCredential holds the login hint, and the
// permission cause, for a platform build the registry refuses. Before, each
// row got "Pin a published build in <dir>/cue.mod/module.cue, then try
// again" and the validation cause. The hint names the host when the registry
// mapping holds exactly one, and is the bare command when it holds two.
func TestPlatformBuildHint_RefusedCredential(t *testing.T) {
	sole := func(registry string) string { return hintLogin + " " + registry }
	bare := func(string) string { return hintLogin }
	for _, tc := range []struct {
		name     string
		registry func(t *testing.T) string
		contains string
		want     func(registry string) string
	}{
		{
			name:     "registry answers 401",
			registry: func(t *testing.T) string { return cuemodtest.StatusRegistry(t, http.StatusUnauthorized) },
			contains: "401 Unauthorized",
			want:     sole,
		},
		{"archive blob answers 401", blobAnswers(http.StatusUnauthorized), "401 Unauthorized", sole},
		{"archive blob answers 403", blobAnswers(http.StatusForbidden), "403 Forbidden", sole},
		{
			name: "registry mapping with two hosts",
			registry: func(t *testing.T) string {
				return "example.com/dep=" + cuemodtest.StatusRegistry(t, http.StatusUnauthorized) + ",registry.invalid"
			},
			contains: "401 Unauthorized",
			want:     bare,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOCKER_CONFIG", t.TempDir())
			cuemodtest.ColdCache(t)
			dir := hintPlatform(t, depPinned, importsDep)
			registry := tc.registry(t)
			_, err := BuildPlatformModule(context.Background(), dir, registry)
			require.Error(t, err)
			require.ErrorIs(t, err, oerrors.ErrPermission)
			require.NotErrorIs(t, err, oerrors.ErrValidation)
			var detail *oerrors.DetailError
			require.True(t, errors.As(err, &detail), "%v", err)
			assert.Contains(t, detail.Message, tc.contains, "the registry's own answer stays in the message")
			assert.Equal(t, tc.want(registry), detail.Hint, "%s", detail.Message)
		})
	}
}

// TestPlatformBuildHint_RefusalNotTypedAsOne_Pinned records known gaps, it
// does not state the wanted answer. Each registry here refuses the caller,
// yet the refusal reaches the cli typed as something else, so the build keeps
// the pin hint and the validation cause: CUE's registry client reports a 403
// answer to the tag lookup as "module not found", a token endpoint that
// answers 403 ends the same way, and a token endpoint that answers 401 fails
// with "cannot do HTTP request: ...: 401 Unauthorized", which the library
// reads as no response. The wanted answer for all three is the login hint
// and the permission cause. The reading of registry error text is the
// library's alone; when it types one of these as a refusal, its row fails:
// move it to TestPlatformBuildHint_RefusedCredential.
func TestPlatformBuildHint_RefusalNotTypedAsOne_Pinned(t *testing.T) {
	for _, tc := range []struct {
		name     string
		registry func(t *testing.T) string
		contains string
	}{
		{"tag lookup answers 403", func(t *testing.T) string { return cuemodtest.StatusRegistry(t, http.StatusForbidden) }, "module not found"},
		{"token endpoint answers 403", func(t *testing.T) string { return cuemodtest.TokenRegistry(t, http.StatusForbidden) }, "module not found"},
		{"token endpoint answers 401", func(t *testing.T) string { return cuemodtest.TokenRegistry(t, http.StatusUnauthorized) }, "401 Unauthorized"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOCKER_CONFIG", t.TempDir())
			cuemodtest.ColdCache(t)
			dir := hintPlatform(t, depPinned, importsDep)
			_, err := BuildPlatformModule(context.Background(), dir, tc.registry(t))
			require.Error(t, err)
			require.ErrorIs(t, err, oerrors.ErrValidation, "the gap closed: move the row to TestPlatformBuildHint_RefusedCredential")
			var detail *oerrors.DetailError
			require.True(t, errors.As(err, &detail), "%v", err)
			assert.Contains(t, detail.Message, tc.contains, "what the user sees of the registry's answer")
			assert.Contains(t, detail.Hint, hintPin, "the gap closed: move the row to TestPlatformBuildHint_RefusedCredential")
		})
	}
}

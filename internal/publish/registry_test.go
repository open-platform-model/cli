package publish

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"cuelabs.dev/go/oci/ociregistry/ocimem"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/mod/modregistrytest"
	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pushFixture runs the pipeline to GO and pushes, returning the plan and the
// options used.
func pushFixture(t *testing.T, registry string, files map[string]string) (*Plan, Options) {
	t.Helper()
	dir := writeTree(t, files)
	var opts Options
	p := runDir(t, KindModule, dir, func(o *Options) {
		o.Registry = registry
		opts = *o
	})
	require.True(t, p.Go(), refusalHeadlines(p))
	require.NoError(t, Push(context.Background(), opts, p))
	return p, opts
}

func TestGate_AlreadyPublished(t *testing.T) {
	registry := emptyTestRegistry(t)
	files := moduleFiles()
	_, opts := pushFixture(t, registry, files)

	// The same tree at the same version: the lookup finds the tag and
	// refuses. No flag or mode turns this into a success — there is none to
	// pass.
	p2, err := Run(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, p2.Refusals, 1, refusalHeadlines(p2))
	r := p2.Refusals[0]
	assert.Contains(t, r.Headline, "example.com/modules/demo@v1 already holds v1.2.0")
	assert.Contains(t, r.Consequence, "fixed bytes permanently")
	assert.Contains(t, r.Action, "opm module version set 1.2.1")
}

func TestGate_AlreadyPublished_UnreachableRegistryIsConnectivity(t *testing.T) {
	// A port nothing listens on: the lookup fails to connect, which is a
	// connectivity error, never a refusal.
	unreachable := func(t *testing.T, files map[string]string) (*Plan, error) {
		t.Helper()
		dir := writeTree(t, files)
		cueCtx := cuecontext.New()
		return Run(context.Background(), Options{
			Dir:            dir,
			Kind:           KindModule,
			Context:        cueCtx,
			IdentitySchema: stubSchema(t, cueCtx),
			Registry:       "127.0.0.1:1+insecure",
			// The override refusal below must not be waived.
		})
	}

	t.Run("clean tree renders INCOMPLETE, never GO", func(t *testing.T) {
		p, err := unreachable(t, moduleFiles())
		require.Error(t, err)
		var connErr *ConnectivityError
		require.ErrorAs(t, err, &connErr)
		assert.Contains(t, connErr.Error(), "listing published versions")

		// The plan is still returned and renders an incomplete verdict — a
		// GO here would be one the gates never finished earning.
		require.NotNil(t, p)
		assert.True(t, p.Go())
		assert.False(t, p.RegistryChecked)
		assert.Contains(t, p.Render(), "INCOMPLETE")
		assert.NotContains(t, p.Render(), "GO —")
	})

	t.Run("locally-derived refusals survive the connectivity failure", func(t *testing.T) {
		files := edit(moduleFiles(), "cue.mod/module.cue", `module: "example.com/modules/demo@v1"
language: version: "v0.17.0"
source: kind: "self"
deps: "example.com/dep@v1": v: "v1.4.0"
`)
		files = edit(files, "cue.mod/local-module.cue", `deps: "example.com/dep@v1": replaceWith: "../dep"
`)
		p, err := unreachable(t, files)
		require.Error(t, err)
		require.NotNil(t, p)
		// The offline author still sees the problem they can fix right now.
		assert.Contains(t, refusalHeadlines(p), "local-module.cue is present")
		assert.Contains(t, p.Render(), "REFUSED")
	})
}

// TestRun_NonexistentDirIsAnError: a typo'd path must be a plain not-found
// error — never refusal msg 3, whose "opm mod init" action cannot help.
func TestRun_NonexistentDirIsAnError(t *testing.T) {
	cueCtx := cuecontext.New()
	opts := Options{
		Dir:            filepath.Join(t.TempDir(), "no-such-module"),
		Kind:           KindModule,
		Context:        cueCtx,
		IdentitySchema: stubSchema(t, cueCtx),
	}

	p, err := Run(context.Background(), opts)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "does not exist")

	_, _, err = VetChecks(opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

// TestPush_OverrideNeverBakedIn pins the mechanism behind D6's "replacements
// are ignored either way": modzip omits cue.mod/local-module.cue from the
// created zip, so the published artifact can only resolve published deps.
func TestPush_OverrideNeverBakedIn(t *testing.T) {
	registry := emptyTestRegistry(t)
	files := edit(moduleFiles(), "cue.mod/module.cue", `module: "example.com/modules/demo@v1"
language: version: "v0.17.0"
source: kind: "self"
deps: "example.com/dep@v1": v: "v1.4.0"
`)
	files = edit(files, "cue.mod/local-module.cue", `deps: "example.com/dep@v1": replaceWith: "../dep"
`)
	dir := writeTree(t, files)
	var opts Options
	p := runDir(t, KindModule, dir, func(o *Options) {
		o.Registry = registry
		o.SkipOverrideCheck = true
		opts = *o
	})
	require.True(t, p.Go(), refusalHeadlines(p))
	require.True(t, p.OverrideWaived)
	require.NoError(t, Push(context.Background(), opts, p))

	client, err := NewRegistryClient(opts.Registry)
	require.NoError(t, err)
	mv, err := module.NewVersion(p.DeclaredPath, p.Tag)
	require.NoError(t, err)
	m, err := client.GetModule(context.Background(), mv)
	require.NoError(t, err)

	names := publishedNames(t, m.GetZip)
	assert.NotContains(t, names, "cue.mod/local-module.cue")
	// The published cue.mod still pins the registry dependency the artifact
	// will resolve.
	modFile := publishedFile(t, m.GetZip, "cue.mod/module.cue")
	assert.Contains(t, modFile, `"example.com/dep@v1"`)
	assert.Contains(t, modFile, "v1.4.0")
}

// TestPush_RoundTrip publishes and resolves the module back: the published
// bytes are the committed tree (D2), identity file included.
func TestPush_RoundTrip(t *testing.T) {
	registry := emptyTestRegistry(t)
	files := moduleFiles()
	p, opts := pushFixture(t, registry, files)

	client, err := NewRegistryClient(opts.Registry)
	require.NoError(t, err)
	mv, err := module.NewVersion(p.DeclaredPath, p.Tag)
	require.NoError(t, err)

	m, err := client.GetModule(context.Background(), mv)
	require.NoError(t, err)

	modFile, err := m.ModuleFile(context.Background())
	require.NoError(t, err)
	assert.Contains(t, string(modFile), `"example.com/modules/demo@v1"`)

	identity := publishedFile(t, m.GetZip, "identity/identity.cue")
	assert.Equal(t, files["identity/identity.cue"], identity)
}

// TestPush_WritesVersionFillBeforeZipping: an open Version filled by
// --version reaches both the working tree and the published bytes — an
// in-memory overlay would publish bytes the tree does not hold.
func TestPush_WritesVersionFillBeforeZipping(t *testing.T) {
	registry := emptyTestRegistry(t)
	files := edit(moduleFiles(), "identity/identity.cue", `package identity

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+"

ModulePath: "example.com/modules/demo@v1"
Version:    #VersionType
`)
	files = edit(files, "module.cue", `package demo

metadata: {
	name:       "demo"
	modulePath: "example.com/modules/demo@v1"
}
`)
	dir := writeTree(t, files)
	var opts Options
	p := runDir(t, KindModule, dir, func(o *Options) {
		o.Registry = registry
		o.VersionFlag = "1.3.0"
		opts = *o
	})
	require.True(t, p.Go(), refusalHeadlines(p))
	require.Equal(t, "1.3.0", p.FillVersion)
	require.NoError(t, Push(context.Background(), opts, p))

	// The working tree was written (D12)…
	onDisk, err := os.ReadFile(filepath.Join(dir, "identity", "identity.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(onDisk), `#VersionType & "1.3.0"`)

	// …and the published bytes are that written tree.
	client, err := NewRegistryClient(opts.Registry)
	require.NoError(t, err)
	mv, err := module.NewVersion("example.com/modules/demo@v1", "v1.3.0")
	require.NoError(t, err)
	m, err := client.GetModule(context.Background(), mv)
	require.NoError(t, err)
	published := publishedFile(t, m.GetZip, "identity/identity.cue")
	assert.Equal(t, string(onDisk), published)
}

// TestPush_Authenticated pushes through modconfig's credential chain against
// a registry requiring basic auth — push and pull share one resolver path
// (D11), so `docker login` is all CI needs.
func TestPush_Authenticated(t *testing.T) {
	reg, err := modregistrytest.NewServer(
		ocimem.NewWithConfig(&ocimem.Config{ImmutableTags: true}),
		&modregistrytest.AuthConfig{Username: "opm", Password: "sesame"},
	)
	require.NoError(t, err)
	t.Cleanup(reg.Close)

	// Credentials arrive the way docker login leaves them.
	dockerDir := t.TempDir()
	auth := base64.StdEncoding.EncodeToString([]byte("opm:sesame"))
	cfg := map[string]any{"auths": map[string]any{reg.Host(): map[string]string{"auth": auth}}}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dockerDir, "config.json"), data, 0o600))
	t.Setenv("DOCKER_CONFIG", dockerDir)

	registry := reg.Host() + "+insecure"
	p, _ := pushFixture(t, registry, moduleFiles())
	assert.Equal(t, "v1.2.0", p.Tag)
}

// publishedFile reads one file out of a published module zip.
func publishedFile(t *testing.T, getZip func(context.Context) (io.ReadCloser, error), name string) string {
	t.Helper()
	rc, err := getZip(context.Background())
	require.NoError(t, err)
	defer rc.Close()
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		r, err := f.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(r)
		require.NoError(t, r.Close())
		require.NoError(t, err)
		return string(content)
	}
	t.Fatalf("file %s not found in published zip (have %v)", name, fileNames(zr))
	return ""
}

// publishedNames lists every file name in a published module zip.
func publishedNames(t *testing.T, getZip func(context.Context) (io.ReadCloser, error)) []string {
	t.Helper()
	rc, err := getZip(context.Background())
	require.NoError(t, err)
	defer rc.Close()
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	return fileNames(zr)
}

func fileNames(zr *zip.Reader) []string {
	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	return names
}

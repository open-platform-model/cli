package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/controller/controllertest"
	"github.com/open-platform-model/cli/internal/controller/crds"
	"github.com/open-platform-model/cli/internal/modref"
)

// pinRegistry serves three controller module versions: 0.1.0 and 0.2.0 deploy
// beta releases of 1.0, 0.3.0 deploys 1.1.0, 0.4.0 states no controller.
func pinRegistry(t *testing.T) string {
	t.Helper()
	return controllertest.Registry(t,
		controllertest.Version{Module: "0.1.0", Controller: "1.0.0-beta.5"},
		controllertest.Version{Module: "0.2.0", Controller: "1.0.0-beta.6"},
		controllertest.Version{Module: "0.3.0", Controller: "1.1.0"},
		controllertest.Version{Module: "0.4.0", ControllerPackage: "-"},
	)
}

// fakeRender stands in for the module render: each module version renders
// two CRDs whose version label is that version, plus a Deployment the pin
// ignores. A local directory renders as version "local". It records what it
// was asked to render.
type fakeRender struct {
	err   error
	calls []string
}

func (f *fakeRender) render(_ context.Context, _ string, res *modref.Resolution, dir string) ([]*unstructured.Unstructured, error) {
	version := "local"
	if res != nil {
		version = res.Version
	}
	f.calls = append(f.calls, version+"|"+dir)
	if f.err != nil {
		return nil, f.err
	}
	return []*unstructured.Unstructured{
		fakeCRD("platforms.opmodel.dev", version),
		{Object: map[string]any{"apiVersion": "apps/v1", "kind": "Deployment", "metadata": map[string]any{"name": "opm-controller-manager"}}},
		fakeCRD("moduleinstances.opmodel.dev", version),
	}, nil
}

func fakeCRD(name, version string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       "CustomResourceDefinition",
		"metadata": map[string]any{
			"name":   name,
			"labels": map[string]any{"module.opmodel.dev/version": version},
		},
		"spec": map[string]any{"group": "opmodel.dev", "scope": "Namespaced"},
	}}
}

// manifests is what a render of version writes, by file name.
func manifests(t *testing.T, version string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, name := range []string{"moduleinstances.opmodel.dev", "platforms.opmodel.dev"} {
		data, err := crds.Manifest(fakeCRD(name, version))
		require.NoError(t, err)
		out[name+".yaml"] = string(data)
	}
	return out
}

// testPaths is a pin file and a manifest directory in a temporary directory,
// with the fake render.
func testPaths(t *testing.T, mapping string, f *fakeRender) paths {
	t.Helper()
	dir := t.TempDir()
	crdDir := filepath.Join(dir, "crds")
	require.NoError(t, os.MkdirAll(crdDir, 0o755))
	return paths{mapping: mapping, pin: filepath.Join(dir, "pin.go"), crds: crdDir, render: f.render}
}

// readDir returns the manifest directory's *.yaml files by name.
func readDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	files, err := readManifests(dir)
	require.NoError(t, err)
	out := map[string]string{}
	for k, v := range files {
		out[k] = string(v)
	}
	return out
}

func runTool(t *testing.T, p paths, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(context.Background(), args, &out, &errOut, p)
	return code, out.String(), errOut.String()
}

// "Pin refresh is one reviewable diff": the program writes exactly the
// golden pin file and the CRD manifests of the version's render, and
// replaces the manifests that were there.
func TestPin_WritesThePinAndTheManifests(t *testing.T) {
	f := &fakeRender{}
	p := testPaths(t, pinRegistry(t), f)
	require.NoError(t, os.WriteFile(filepath.Join(p.crds, "dropped.opmodel.dev.yaml"), []byte("old\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(p.crds, "crds.go"), []byte("package crds\n"), 0o644))

	code, stdout, stderr := runTool(t, p, "0.2.0")
	require.Equal(t, 0, code, stderr)
	assert.Contains(t, stdout, "0.2.0 (deploys opm-controller v1.0.0-beta.6; 2 CRDs in ")
	assert.Equal(t, []string{"v0.2.0|"}, f.calls, "the resolved version renders, not a directory")

	got, err := os.ReadFile(p.pin)
	require.NoError(t, err)
	want, err := os.ReadFile(filepath.Join("testdata", "pin.go.golden"))
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))

	assert.Equal(t, manifests(t, "v0.2.0"), readDir(t, p.crds), "the render's CRDs only, the dropped one removed")
	_, err = os.Stat(filepath.Join(p.crds, "crds.go"))
	assert.NoError(t, err, "only manifests are replaced")
}

// "A module without a readable controller version is refused by the pin
// task": non-zero, naming the version, and the pin and the manifests
// unchanged. A render that fails changes nothing either.
func TestPin_RefusedVersionLeavesThePinAndTheManifests(t *testing.T) {
	cases := []struct {
		name    string
		version string
		render  *fakeRender
		want    string
	}{
		{"no controller version", "0.4.0", &fakeRender{}, "0.4.0"},
		{"not served", "0.9.0", &fakeRender{}, "0.9.0"},
		{"render fails", "0.2.0", &fakeRender{err: errors.New("catalog unreachable")}, "rendering opmodel.dev/modules/opm_controller 0.2.0: catalog unreachable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := testPaths(t, pinRegistry(t), tc.render)
			require.NoError(t, os.WriteFile(p.pin, []byte("unchanged\n"), 0o644))
			require.NoError(t, os.WriteFile(filepath.Join(p.crds, "kept.opmodel.dev.yaml"), []byte("kept\n"), 0o644))

			code, _, stderr := runTool(t, p, tc.version)
			assert.Equal(t, 1, code)
			assert.Contains(t, stderr, tc.want)
			got, err := os.ReadFile(p.pin)
			require.NoError(t, err)
			assert.Equal(t, "unchanged\n", string(got))
			assert.Equal(t, map[string]string{"kept.opmodel.dev.yaml": "kept\n"}, readDir(t, p.crds))
		})
	}
}

// A render without a CRD is no pin.
func TestPin_RenderWithoutCRDsIsRefused(t *testing.T) {
	p := testPaths(t, pinRegistry(t), &fakeRender{})
	p.render = func(context.Context, string, *modref.Resolution, string) ([]*unstructured.Unstructured, error) {
		return nil, nil
	}
	code, _, stderr := runTool(t, p, "0.2.0")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "no CustomResourceDefinition")
	_, err := os.Stat(p.pin)
	assert.True(t, os.IsNotExist(err), "no pin written")
}

// --crds-from renders a local module directory and writes only the
// manifests; it needs no registry.
func TestPin_CRDsFromWritesOnlyTheManifests(t *testing.T) {
	f := &fakeRender{}
	p := testPaths(t, "opmodel.dev=127.0.0.1:1+insecure", f)
	require.NoError(t, os.WriteFile(p.pin, []byte("unchanged\n"), 0o644))

	code, stdout, stderr := runTool(t, p, "--crds-from", "/src/modules/opm_controller")
	require.Equal(t, 0, code, stderr)
	assert.Contains(t, stdout, "wrote 2 CRDs from /src/modules/opm_controller")
	assert.Equal(t, []string{"local|/src/modules/opm_controller"}, f.calls)
	assert.Equal(t, manifests(t, "local"), readDir(t, p.crds))
	got, err := os.ReadFile(p.pin)
	require.NoError(t, err)
	assert.Equal(t, "unchanged\n", string(got))
}

func TestPin_Check(t *testing.T) {
	reg := pinRegistry(t)
	// pinned is a pin file and the manifests of renderedAs, in their own
	// directory.
	pinned := func(f *fakeRender, module, op, renderedAs string) paths {
		p := testPaths(t, reg, f)
		data, err := renderPin(module, op)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(p.pin, data, 0o644))
		for name, body := range manifests(t, renderedAs) {
			require.NoError(t, os.WriteFile(filepath.Join(p.crds, name), []byte(body), 0o644))
		}
		return p
	}

	code, stdout, stderr := runTool(t, pinned(&fakeRender{}, "0.2.0", "v1.0.0-beta.6", "v0.2.0"), "--check")
	assert.Equal(t, 0, code, stderr)
	assert.Contains(t, stdout, "consistent")

	// The recorded controller version is out of step with the module.
	code, _, stderr = runTool(t, pinned(&fakeRender{}, "0.2.0", "v1.0.0-beta.5", "v0.2.0"), "--check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "v1.0.0-beta.5")
	assert.Contains(t, stderr, "v1.0.0-beta.6")
	assert.Contains(t, stderr, "task controller:pin VERSION=0.2.0")

	// The manifests are another version's render: each one differs.
	code, _, stderr = runTool(t, pinned(&fakeRender{}, "0.2.0", "v1.0.0-beta.6", "v0.1.0"), "--check")
	assert.Equal(t, 1, code)
	assert.Equal(t, 2, strings.Count(stderr, "differs from the CRD opmodel.dev/modules/opm_controller 0.2.0 renders; run 'task controller:pin VERSION=0.2.0'"), stderr)

	// A missing and an extra manifest are named.
	p := pinned(&fakeRender{}, "0.2.0", "v1.0.0-beta.6", "v0.2.0")
	require.NoError(t, os.Remove(filepath.Join(p.crds, "platforms.opmodel.dev.yaml")))
	require.NoError(t, os.WriteFile(filepath.Join(p.crds, "extra.opmodel.dev.yaml"), []byte("x\n"), 0o644))
	code, _, stderr = runTool(t, p, "--check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "platforms.opmodel.dev.yaml: missing")
	assert.Contains(t, stderr, "extra.opmodel.dev.yaml: opmodel.dev/modules/opm_controller 0.2.0 renders no such CRD")

	// A render that fails is a failure to check, not a pass.
	code, _, stderr = runTool(t, pinned(&fakeRender{err: errors.New("catalog unreachable")}, "0.2.0", "v1.0.0-beta.6", "v0.2.0"), "--check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "catalog unreachable")

	// The pinned module version is not served.
	code, _, stderr = runTool(t, pinned(&fakeRender{}, "0.9.0", "v1.0.0", "v0.9.0"), "--check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "pin.go")
	assert.Contains(t, stderr, "opmodel.dev/modules/opm_controller 0.9.0 is not served")

	// An unreadable pin is a failure, not a pass.
	p = testPaths(t, reg, &fakeRender{})
	code, _, stderr = runTool(t, p, "--check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "reading the pin")

	// A registry that cannot be reached is a failure, not a pass.
	p = pinned(&fakeRender{}, "0.2.0", "v1.0.0-beta.6", "v0.2.0")
	p.mapping = "opmodel.dev=127.0.0.1:1+insecure"
	code, _, stderr = runTool(t, p, "--check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "looking up")
}

func TestPin_Select(t *testing.T) {
	p := testPaths(t, pinRegistry(t), &fakeRender{})

	// The newest at or below the bound whose controller the cli can drive.
	code, stdout, stderr := runTool(t, p, "select", "0.3.0", "1.0.0-beta.9")
	assert.Equal(t, 0, code, stderr)
	assert.Equal(t, "0.2.0\n", stdout)

	code, stdout, _ = runTool(t, p, "select", "0.3.0", "1.1.0")
	assert.Equal(t, 0, code)
	assert.Equal(t, "0.3.0\n", stdout)

	code, stdout, _ = runTool(t, p, "select", "0.2.0", "1.1.0")
	assert.Equal(t, 0, code)
	assert.Equal(t, "0.2.0\n", stdout)

	code, stdout, _ = runTool(t, p, "select", "0.3.0", "0.9.0")
	assert.Equal(t, exitNone, code)
	assert.Empty(t, stdout)
}

func TestPin_Usage(t *testing.T) {
	code, _, stderr := runTool(t, testPaths(t, "opmodel.dev=127.0.0.1:1+insecure", &fakeRender{}), "--bogus")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "usage: controller-pin")
}

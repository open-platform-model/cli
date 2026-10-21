// Package controllertest serves versions of the controller module from an
// in-memory registry, so tests of the controller-version reader, the pin tool
// and the install path run offline. Only tests import it.
package controllertest

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/require"
)

// ModulePath is the controller module's path without its major; it equals
// controller.ControllerModulePath (not imported here, to keep this package a
// leaf every test can import).
const ModulePath = "opmodel.dev/modules/opm_controller"

// Version is one published controller module version.
type Version struct {
	// Module is the module version, bare ("0.1.0").
	Module string
	// Controller is the controller.Version the module states, bare
	// ("1.0.0-beta.7"). Ignored when ControllerPackage is set.
	Controller string
	// ControllerPackage, when set, replaces the controller package's file
	// content; "-" leaves the package out.
	ControllerPackage string
}

// ControllerFile is the controller package as the controller module ships it.
func ControllerFile(controllerVersion string) string {
	return fmt.Sprintf(`package controller

Version: %q

Image: {
	repository: "ghcr.io/open-platform-model/opm-controller"
	tag:        "v\(Version)"
	digest:     "sha256:0000000000000000000000000000000000000000000000000000000000000000"
}
`, controllerVersion)
}

// Registry serves the given versions of the controller module and returns the
// CUE registry mapping that routes opmodel.dev to it. It points
// CUE_CACHE_DIR at a temporary directory for the test, so nothing reaches
// the user's module cache, and names no fallback registry, so a lookup that
// escapes the mapping fails instead of reaching the network.
func Registry(t *testing.T, versions ...Version) string {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("CUE_CACHE_DIR", cache)
	// CUE extracts fetched modules read-only; make them removable before
	// the temporary directory's own cleanup runs.
	t.Cleanup(func() { makeWritable(cache) })
	fsys := fstest.MapFS{}
	for _, v := range versions {
		major := "v" + strings.SplitN(v.Module, ".", 2)[0]
		dir := strings.ReplaceAll(ModulePath, "/", "_") + "_v" + v.Module
		fsys[dir+"/cue.mod/module.cue"] = &fstest.MapFile{Data: []byte(fmt.Sprintf(
			"module: %q\nlanguage: version: \"v0.17.0\"\n", ModulePath+"@"+major))}
		fsys[dir+"/module.cue"] = &fstest.MapFile{Data: []byte("package opm_controller\n")}
		fsys[dir+"/identity/identity.cue"] = &fstest.MapFile{Data: []byte(fmt.Sprintf(
			"package identity\n\nModulePath: %q\nVersion: %q\n", ModulePath+"@"+major, v.Module))}
		switch {
		case v.ControllerPackage == "-":
		case v.ControllerPackage != "":
			fsys[dir+"/controller/controller.cue"] = &fstest.MapFile{Data: []byte(v.ControllerPackage)}
		default:
			fsys[dir+"/controller/controller.cue"] = &fstest.MapFile{Data: []byte(ControllerFile(v.Controller))}
		}
	}
	reg, err := modregistrytest.New(fsys, "")
	require.NoError(t, err)
	t.Cleanup(reg.Close)
	return "opmodel.dev=" + reg.Host() + "+insecure"
}

// Source returns modconfig's cached registry over a mapping from Registry,
// the type install and the pin tool fetch the module through.
func Source(t *testing.T, mapping string) modconfig.CachedRegistry {
	t.Helper()
	src, err := modconfig.NewRegistry(&modconfig.Config{CUERegistry: mapping})
	require.NoError(t, err)
	return src
}

// makeWritable adds owner write permission to every directory under root.
func makeWritable(root string) {
	//nolint:errcheck // best effort: a leftover file only fails the cleanup
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			os.Chmod(p, 0o755) //nolint:errcheck,gosec // test cache directory; best effort
		}
		return nil
	})
}

// Package operatortest serves versions of the operator module from an
// in-memory registry, so tests of the operator-version reader, the pin tool
// and the install path run offline. Only tests import it.
package operatortest

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

// ModulePath is the operator module's path without its major; it equals
// operator.OperatorModulePath (not imported here, to keep this package a
// leaf every test can import).
const ModulePath = "opmodel.dev/modules/opm_operator"

// Version is one published operator module version.
type Version struct {
	// Module is the module version, bare ("0.1.0").
	Module string
	// Operator is the operator.Version the module states, bare
	// ("1.0.0-beta.7"). Ignored when OperatorPackage is set.
	Operator string
	// OperatorPackage, when set, replaces the operator package's file
	// content; "-" leaves the package out.
	OperatorPackage string
}

// OperatorFile is the operator package as the operator module ships it.
func OperatorFile(operatorVersion string) string {
	return fmt.Sprintf(`package operator

Version: %q

Image: {
	repository: "ghcr.io/open-platform-model/opm-operator"
	tag:        "v\(Version)"
	digest:     "sha256:0000000000000000000000000000000000000000000000000000000000000000"
}
`, operatorVersion)
}

// Registry serves the given versions of the operator module and returns the
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
		fsys[dir+"/module.cue"] = &fstest.MapFile{Data: []byte("package opm_operator\n")}
		fsys[dir+"/identity/identity.cue"] = &fstest.MapFile{Data: []byte(fmt.Sprintf(
			"package identity\n\nModulePath: %q\nVersion: %q\n", ModulePath+"@"+major, v.Module))}
		switch {
		case v.OperatorPackage == "-":
		case v.OperatorPackage != "":
			fsys[dir+"/operator/operator.cue"] = &fstest.MapFile{Data: []byte(v.OperatorPackage)}
		default:
			fsys[dir+"/operator/operator.cue"] = &fstest.MapFile{Data: []byte(OperatorFile(v.Operator))}
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

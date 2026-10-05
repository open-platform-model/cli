package publish

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
)

// The pins in this file record what fetchPublishedTree and
// loadPublishedPackage answer for each way a registry fetch fails. They were
// measured against the text probes these functions used before the library
// classified registry failures, and must hold unchanged across that swap:
// each answer is an exit code (ErrNotPublished 5, *ConnectivityError 3,
// found=false the compat walk's negative signal).

const pinDepModuleCue = `module: "example.com/dep@v0"
language: version: "v0.9.0"
`

// pinPredecessor is a published build example.com/cat@v1 v1.0.0 that depends
// on example.com/dep@v0 v0.1.0. Package pkg imports imp; package plain
// imports nothing. Directory parent holds only a subpackage, docs holds only
// a README, and hidden holds only a .cue file cue/load ignores: none is a
// package at this version.
func pinPredecessor(imp string) map[string]string {
	return map[string]string{
		"cue.mod/module.cue": `module: "example.com/cat@v1"
language: version: "v0.9.0"
deps: "example.com/dep@v0": v: "v0.1.0"
`,
		"root.cue":           "package cat\n",
		"pkg/pkg.cue":        "package pkg\n\nimport d \"" + imp + "\"\n\nx: d\n",
		"plain/plain.cue":    "package plain\n\nx: 1\n",
		"parent/sub/sub.cue": "package sub\n\nx: 1\n",
		"docs/README.md":     "not CUE\n",
		"hidden/_x.cue":      "package hidden\n\nx: 1\n",
	}
}

// pinRegistry pushes the predecessor importing imp and, when withDep, its
// dependency, and returns the registry.
func pinRegistry(t *testing.T, imp string, withDep bool) string {
	t.Helper()
	reg := emptyTestRegistry(t)
	if withDep {
		rawPush(t, reg, map[string]string{"cue.mod/module.cue": pinDepModuleCue, "dep.cue": "package dep\n\nv: 1\n"}, "example.com/dep@v0", "v0.1.0")
	}
	rawPush(t, reg, pinPredecessor(imp), "example.com/cat@v1", "v1.0.0")
	return reg
}

func healthy(t *testing.T) string { return pinRegistry(t, "example.com/dep@v0", true) }

func fronted(answer cuemodtest.Answer) func(t *testing.T) string {
	return func(t *testing.T) string { return cuemodtest.Fronted(t, healthy(t), answer) }
}

func statusOnly(code int) func(t *testing.T) string {
	return func(t *testing.T) string { return cuemodtest.StatusRegistry(t, code) }
}

func refused(*testing.T) string { return cuemodtest.UnreachableRegistry }

// answer names what a call answered, in exit-code terms.
func answer(found bool, err error) string {
	var connErr *ConnectivityError
	switch {
	case err == nil && found:
		return "found"
	case err == nil:
		return "absent"
	case errors.Is(err, ErrNotPublished):
		return "not published"
	case errors.As(err, &connErr):
		return "connectivity"
	default:
		return "other"
	}
}

func TestFetchPublishedTree_Pinned(t *testing.T) {
	for _, tc := range []struct {
		name     string
		registry func(t *testing.T) string
		version  string
		want     string
	}{
		{"version not held", healthy, "v1.9.0", "not published"},
		{"403 on every request", statusOnly(http.StatusForbidden), "v1.0.0", "not published"},
		{"401", statusOnly(http.StatusUnauthorized), "v1.0.0", "connectivity"},
		{"429", statusOnly(http.StatusTooManyRequests), "v1.0.0", "connectivity"},
		{"503", statusOnly(http.StatusServiceUnavailable), "v1.0.0", "connectivity"},
		{"refused connection", refused, "v1.0.0", "connectivity"},
		{"tag held, archive blob 404", fronted(cuemodtest.BlobAnswers("example.com/cat", http.StatusNotFound)), "v1.0.0", "connectivity"},
		{"registry mapping does not parse", func(*testing.T) string { return "::nonsense::" }, "v1.0.0", "other"},
		{"held", healthy, "v1.0.0", "found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			coldCUECache(t)
			dir, err := fetchPublishedTree(context.Background(), tc.registry(t), "example.com/cat", tc.version)
			assert.Equal(t, tc.want, answer(dir != "", err), "%v", err)
		})
	}
}

func TestLoadPublishedPackage_Pinned(t *testing.T) {
	for _, tc := range []struct {
		name     string
		registry func(t *testing.T) string
		pkg      string
		version  string
		want     string
	}{
		{"present", healthy, "pkg", "v1.0.0", "found"},
		{"version not held", healthy, "pkg", "v1.9.0", "absent"},
		{"package absent at a held version", healthy, "nothere", "v1.0.0", "absent"},
		{"directory holds only a subpackage", healthy, "parent", "v1.0.0", "absent"},
		{"directory holds only a README", healthy, "docs", "v1.0.0", "absent"},
		// cue/load reports a directory whose only .cue file it ignores as
		// "no files in package directory", which no registry answer caused
		// and which the walk has always reported as connectivity.
		{"directory holds only an ignored .cue file", healthy, "hidden", "v1.0.0", "connectivity"},
		{"403 on the probed repository", fronted(cuemodtest.RepoAnswers("example.com/cat", http.StatusForbidden)), "pkg", "v1.0.0", "absent"},
		{"dependency not held", func(t *testing.T) string { return pinRegistry(t, "example.com/dep@v0", false) }, "pkg", "v1.0.0", "connectivity"},
		{"dependency answered 403", fronted(cuemodtest.RepoAnswers("example.com/dep", http.StatusForbidden)), "pkg", "v1.0.0", "connectivity"},
		{"dependency archive blob 404", fronted(cuemodtest.BlobAnswers("example.com/dep", http.StatusNotFound)), "pkg", "v1.0.0", "connectivity"},
		// An import that no module of the build provides is an author
		// defect, not a registry failure; today it reads as absent.
		{"import its dependency does not provide", func(t *testing.T) string { return pinRegistry(t, "example.com/dep/missing@v0", true) }, "pkg", "v1.0.0", "absent"},
		{"import of a missing own-path package", func(t *testing.T) string { return pinRegistry(t, "example.com/cat/nothere@v1", true) }, "pkg", "v1.0.0", "absent"},
		{"probed archive blob 404", fronted(cuemodtest.BlobAnswers("example.com/cat", http.StatusNotFound)), "pkg", "v1.0.0", "connectivity"},
		{"401 on the probed repository", fronted(cuemodtest.RepoAnswers("example.com/cat", http.StatusUnauthorized)), "pkg", "v1.0.0", "connectivity"},
		{"429 on the probed repository", fronted(cuemodtest.RepoAnswers("example.com/cat", http.StatusTooManyRequests)), "pkg", "v1.0.0", "connectivity"},
		{"503 on the probed repository", fronted(cuemodtest.RepoAnswers("example.com/cat", http.StatusServiceUnavailable)), "pkg", "v1.0.0", "connectivity"},
		{"refused connection", refused, "pkg", "v1.0.0", "connectivity"},
		{"registry mapping does not parse", func(*testing.T) string { return "::nonsense::" }, "pkg", "v1.0.0", "connectivity"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			coldCUECache(t)
			opts := Options{Context: cuecontext.New(), Registry: tc.registry(t)}
			_, found, err := loadPublishedPackage(opts, t.TempDir(), "example.com/cat", tc.pkg, tc.version)
			assert.Equal(t, tc.want, answer(found, err), "%v", err)
		})
	}
}

// TestCompatScan_PredecessorDependencyNotHeld runs the walk over a
// predecessor whose dependency the registry does not hold: the walk aborts
// as connectivity (exit 3), whatever compared before it, and no verdict is
// reached.
func TestCompatScan_PredecessorDependencyNotHeld(t *testing.T) {
	coldCUECache(t)
	registry := emptyTestRegistry(t)
	pred := memberCatalogFilesAt("1.0.0")
	pred["cue.mod/module.cue"] += "deps: \"example.com/dep@v0\": v: \"v0.1.0\"\n"
	thing := "resources/v1beta1/thing.cue"
	pred[thing] = strings.Replace(pred[thing], "package v1beta1\n", "package v1beta1\n\nimport d \"example.com/dep@v0\"\n\n_dep: d\n", 1)
	rawPush(t, registry, pred, "example.com/catalogs/demo@v1", "v1.0.0")

	dir := writeTree(t, memberCatalogFilesAt("1.2.0"))
	opts := baseOptions(t, dir)
	opts.Registry = registry
	members, refusals := enumerateMembers(opts, dir)
	require.Empty(t, refusals)

	var g CatalogGateOutcomes
	var verdicts []Refusal
	err := compatScan(opts, "example.com/catalogs/demo", []string{"v1.0.0"}, false, members, &g, func(r Refusal) { verdicts = append(verdicts, r) })
	var connErr *ConnectivityError
	require.ErrorAs(t, err, &connErr)
	assert.Empty(t, verdicts, "members that compared before the abort refuse nothing")
}

// TestProbedPackageAbsent settles a not-found by the probed build itself. A
// fetch that fails without asking the registry (the mapping does not build)
// proves nothing: it reports neither absent nor an error of its own, so the
// caller keeps the load's own *ConnectivityError.
func TestProbedPackageAbsent(t *testing.T) {
	for _, tc := range []struct {
		name       string
		registry   func(t *testing.T) string
		pkg        string
		version    string
		wantAbsent bool
		wantConn   bool
	}{
		{"package present", healthy, "pkg", "v1.0.0", false, false},
		{"package absent at a held version", healthy, "nothere", "v1.0.0", true, false},
		{"directory holds only a subpackage", healthy, "parent", "v1.0.0", true, false},
		{"directory holds only a README", healthy, "docs", "v1.0.0", true, false},
		{"directory holds only an ignored .cue file", healthy, "hidden", "v1.0.0", true, false},
		{"version not held", healthy, "pkg", "v1.9.0", true, false},
		{"refused connection", refused, "pkg", "v1.0.0", false, true},
		{"registry mapping does not parse", func(*testing.T) string { return "::nonsense::" }, "pkg", "v1.0.0", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			coldCUECache(t)
			absent, err := probedPackageAbsent(Options{Registry: tc.registry(t)}, "example.com/cat", tc.pkg, tc.version)
			assert.Equal(t, tc.wantAbsent, absent)
			var connErr *ConnectivityError
			assert.Equal(t, tc.wantConn, errors.As(err, &connErr), "%v", err)
			if !tc.wantConn {
				assert.NoError(t, err)
			}
		})
	}
}

// TestUnprovidedImport_RegistryFailureIsNotAbsent holds the guard that keeps
// a registry failure from reading as absent: the unversioned import text
// counts only when the library's classification finds no registry failure.
func TestUnprovidedImport_RegistryFailureIsNotAbsent(t *testing.T) {
	const unprovided = "cannot find module providing package example.com/x"
	assert.True(t, unprovidedImport(errors.New(unprovided)))
	assert.False(t, unprovidedImport(errors.New(unprovided+": cannot do HTTP request: dial tcp: connection refused")))
	assert.False(t, unprovidedImport(errors.New(unprovided+": GET /v2/x: 503 Service Unavailable: busy")))
}

package modref

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/publish"
)

const webApp = "example.com/modules/web_app"

func resolve(t *testing.T, src Source, sel Selector) (*Resolution, error) {
	t.Helper()
	return Resolve(context.Background(), src, Request{Path: webApp, Selector: sel, CoreMajor: "v2", Registry: "test-registry"})
}

func TestResolve_Float(t *testing.T) {
	src := testRegistry(t,
		testModule{webApp + "@v1", "v1.0.3", "v2"},
		testModule{webApp + "@v1", "v1.0.4", "v2"},
		testModule{webApp + "@v1", "v1.1.0-alpha.1", "v2"},
		testModule{webApp + "@v2", "v2.0.0-alpha.1", "v2"},
		testModule{webApp + "@v2", "v2.0.0-alpha.2", "v2"},
		testModule{webApp + "@v2", "v2.0.1-0.dev.4.gdeadbee", "v2"},
		testModule{webApp + "@v3", "v3.0.0-0.dev.1.gabc1234", "v2"},
	)

	t.Run("major float prefers stable", func(t *testing.T) {
		r, err := resolve(t, src, Selector{Major: "v1"})
		require.NoError(t, err)
		assert.Equal(t, "v1.0.4", r.Version)
		assert.Equal(t, FloatMajor, r.Strategy)
		assert.Equal(t, webApp+"@v1", r.Import())
	})
	t.Run("prerelease-only major never floats to a dev build", func(t *testing.T) {
		r, err := resolve(t, src, Selector{Major: "v2"})
		require.NoError(t, err)
		assert.Equal(t, "v2.0.0-alpha.2", r.Version)
	})
	t.Run("dev-only major is refused", func(t *testing.T) {
		_, err := resolve(t, src, Selector{Major: "v3"})
		assert.Contains(t, refusalOf(t, err).Refusal.Headline, "only development builds")
	})
	t.Run("absent major is refused", func(t *testing.T) {
		_, err := resolve(t, src, Selector{Major: "v4"})
		r := refusalOf(t, err)
		assert.Contains(t, r.Refusal.Headline, "no published version in major v4")
		assert.Contains(t, r.Refusal.Details(), "v3, v2, v1")
	})
}

func TestResolve_Exact(t *testing.T) {
	src := testRegistry(t,
		testModule{webApp + "@v1", "v1.0.3", "v2"},
		testModule{webApp + "@v1", "v1.0.4", "v2"},
		testModule{webApp + "@v1", "v1.2.0-0.dev.3.gabc1234", "v2"},
	)

	t.Run("present", func(t *testing.T) {
		r, err := resolve(t, src, Selector{Exact: "1.0.3"})
		require.NoError(t, err)
		assert.Equal(t, "v1.0.3", r.Version)
		assert.Equal(t, "v1", r.Major)
		assert.Equal(t, Exact, r.Strategy)
	})
	t.Run("dev tag may be pinned", func(t *testing.T) {
		r, err := resolve(t, src, Selector{Exact: "1.2.0-0.dev.3.gabc1234"})
		require.NoError(t, err)
		assert.Equal(t, "v1.2.0-0.dev.3.gabc1234", r.Version)
	})
	t.Run("absent names path, version and registry", func(t *testing.T) {
		_, err := resolve(t, src, Selector{Exact: "9.9.9"})
		r := refusalOf(t, err)
		assert.Contains(t, r.Refusal.Headline, webApp)
		assert.Contains(t, r.Refusal.Headline, "9.9.9")
		assert.Contains(t, r.Refusal.Details(), "test-registry")
	})
}

func TestResolve_HighestCompatibleMajor(t *testing.T) {
	t.Run("skips a major on another core major", func(t *testing.T) {
		src := testRegistry(t,
			testModule{webApp + "@v1", "v1.0.4", "v2"},
			testModule{webApp + "@v2", "v2.0.0", "v2"},
			testModule{webApp + "@v2", "v2.1.0", "v2"},
			testModule{webApp + "@v3", "v3.0.0", "v3"},
		)
		r, err := resolve(t, src, Selector{})
		require.NoError(t, err)
		assert.Equal(t, "v2.1.0", r.Version)
		assert.Equal(t, HighestCompatibleMajor, r.Strategy)
		assert.Equal(t, []Skipped{{"v3", "requires core v3"}}, r.Skipped)
	})
	t.Run("skips a major without a core dependency or a selectable release", func(t *testing.T) {
		src := testRegistry(t,
			testModule{webApp + "@v1", "v1.0.4", "v2"},
			testModule{webApp + "@v2", "v2.0.0", ""},
			testModule{webApp + "@v3", "v3.0.0-0.dev.1.gabc1234", "v2"},
		)
		r, err := resolve(t, src, Selector{})
		require.NoError(t, err)
		assert.Equal(t, "v1.0.4", r.Version)
		assert.Equal(t, []Skipped{
			{"v3", "holds only development builds"},
			{"v2", "declares no opmodel.dev/core dependency"},
		}, r.Skipped)
	})
	t.Run("nothing compatible lists every major", func(t *testing.T) {
		src := testRegistry(t,
			testModule{webApp + "@v0", "v0.1.0", "v1"},
			testModule{webApp + "@v1", "v1.0.0", "v3"},
		)
		_, err := resolve(t, src, Selector{})
		r := refusalOf(t, err)
		assert.Contains(t, r.Refusal.Headline, "builds on core v2")
		details := r.Refusal.Details()
		assert.Contains(t, details, "requires core v3")
		assert.Contains(t, details, "requires core v1")
	})
}

func TestResolve_NoVersions(t *testing.T) {
	src := testRegistry(t, testModule{"example.com/modules/other@v0", "v0.1.0", "v2"})
	_, err := resolve(t, src, Selector{})
	assert.Contains(t, refusalOf(t, err).Refusal.Headline, "has no published versions")
}

func TestResolve_UnreachableRegistryIsConnectivity(t *testing.T) {
	src := testSource(t, "127.0.0.1:1+insecure")
	_, err := resolve(t, src, Selector{Major: "v1"})
	var connErr *publish.ConnectivityError
	require.ErrorAs(t, err, &connErr)
	assert.Contains(t, connErr.Error(), webApp)
	assert.Contains(t, connErr.Error(), "test-registry")
}

func TestNewest(t *testing.T) {
	assert.Equal(t, "v1.0.4", Newest([]string{"v1.0.3", "v1.0.4", "v1.1.0-alpha.1"}))
	assert.Equal(t, "v2.0.0-rc.1", Newest([]string{"v2.0.0-alpha.2", "v2.0.0-rc.1", "v2.0.1-0.dev.4.gdeadbee"}))
	assert.Equal(t, "", Newest([]string{"v3.0.0-0.dev.1.gabc1234"}))
	assert.Equal(t, "", Newest(nil))
}

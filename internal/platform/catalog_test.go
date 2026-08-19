package platform

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/publish"
)

func TestSelectCatalogVersion(t *testing.T) {
	tests := []struct {
		name       string
		published  []string
		prerelease bool
		selected   string
		highest    string
	}{
		{
			name:      "stable release wins over prereleases",
			published: []string{"v2.0.0-alpha.3", "v2.0.0", "v2.1.0"},
			selected:  "v2.1.0",
			highest:   "v2.1.0",
		},
		{
			name:      "prerelease-only history selects nothing in release mode",
			published: []string{"v2.0.0-alpha.1", "v2.0.0-alpha.2", "v2.0.0-alpha.3"},
			selected:  "",
			highest:   "v2.0.0-alpha.3",
		},
		{
			name:       "prerelease mode ignores development builds",
			published:  []string{"v2.0.0-alpha.3", "v2.0.0-0.dev.1754899200.g9ea5927"},
			prerelease: true,
			selected:   "v2.0.0-alpha.3",
			highest:    "v2.0.0-alpha.3",
		},
		{
			name:       "development builds are never selectable in prerelease mode",
			published:  []string{"v2.0.0-0.dev.1754899200.g9ea5927", "v2.0.0-0.dev.1754899999.gabcdef0"},
			prerelease: true,
			selected:   "",
			highest:    "v2.0.0-0.dev.1754899999.gabcdef0",
		},
		{
			name:      "development builds are never selectable in release mode",
			published: []string{"v2.0.0-0.dev.1754899200.g9ea5927"},
			selected:  "",
			highest:   "v2.0.0-0.dev.1754899200.g9ea5927",
		},
		{
			name:       "prerelease mode does not fall back to a stable release",
			published:  []string{"v2.0.0", "v2.1.0"},
			prerelease: true,
			selected:   "",
			highest:    "v2.1.0",
		},
		{
			name:       "named prerelease families other than alpha qualify",
			published:  []string{"v2.0.0-alpha.3", "v2.0.0-beta.1", "v2.0.0-rc.1"},
			prerelease: true,
			selected:   "v2.0.0-rc.1",
			highest:    "v2.0.0-rc.1",
		},
		{
			name:      "empty history selects nothing and reports nothing seen",
			published: nil,
			selected:  "",
			highest:   "",
		},
		{
			name:      "unparseable tags are ignored rather than considered",
			published: []string{"latest", "v2.0.0", "not-a-version"},
			selected:  "v2.0.0",
			highest:   "v2.0.0",
		},
		{
			name:      "ordering is computed, not taken from the list order",
			published: []string{"v2.3.0", "v2.10.0", "v2.2.0"},
			selected:  "v2.10.0",
			highest:   "v2.10.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selected, highest := selectCatalogVersion(tt.published, tt.prerelease)
			assert.Equal(t, tt.selected, selected, "selected version")
			assert.Equal(t, tt.highest, highest, "highest version seen")
		})
	}
}

func TestIsNumericIdentifier(t *testing.T) {
	assert.True(t, isNumericIdentifier("0"))
	assert.True(t, isNumericIdentifier("1754899200"))
	assert.False(t, isNumericIdentifier("alpha"))
	assert.False(t, isNumericIdentifier("rc1"))
	assert.False(t, isNumericIdentifier("1a"))
	assert.False(t, isNumericIdentifier(""))
}

func TestCatalogRefusalNamesTheFlagThatClearsIt(t *testing.T) {
	r := catalogRefusal("ghcr.io/example", DefaultCatalogPath, "v2.0.0-alpha.3", false)

	assert.Contains(t, r.Headline, DefaultCatalogPath)
	assert.Contains(t, r.Headline, "no published release")
	assert.Contains(t, r.Action, "--catalog-prerelease")
	assert.Contains(t, r.Details(), "v2.0.0-alpha.3")
	assert.Contains(t, r.Details(), "ghcr.io/example")
}

func TestCatalogRefusalInPrereleaseModeDoesNotSuggestItself(t *testing.T) {
	r := catalogRefusal("ghcr.io/example", DefaultCatalogPath, "", true)

	assert.Contains(t, r.Headline, "no published prerelease")
	assert.NotContains(t, r.Action, "--catalog-prerelease")
	assert.Contains(t, r.Action, "--skip-platform")
	assert.Contains(t, r.Details(), "(no published versions)")

	var err error = &RefusalError{r}
	require.EqualError(t, err, r.Headline)
}

// fakeLister answers a fixed version listing or a fixed error.
type fakeLister struct {
	versions []string
	err      error
	gotPath  string
}

func (f *fakeLister) ModuleVersions(_ context.Context, modulePath string) ([]string, error) {
	f.gotPath = modulePath
	return f.versions, f.err
}

func TestResolveCatalogVersionReturnsBareSemVer(t *testing.T) {
	lister := &fakeLister{versions: []string{"v2.0.0-alpha.3", "v2.0.0", "v2.1.0"}}

	got, err := resolveCatalogVersion(context.Background(), lister, "ghcr.io/example", DefaultCatalogPath, false)

	require.NoError(t, err)
	assert.Equal(t, "2.1.0", got, "a Platform subscription stores a bare SemVer")
	assert.Equal(t, DefaultCatalogPath, lister.gotPath, "the major-suffixed path scopes the listing")
}

func TestResolveCatalogVersionPrereleaseMode(t *testing.T) {
	lister := &fakeLister{versions: []string{"v2.0.0-alpha.3", "v2.0.0-0.dev.1754899200.g9ea5927"}}

	got, err := resolveCatalogVersion(context.Background(), lister, "ghcr.io/example", DefaultCatalogPath, true)

	require.NoError(t, err)
	assert.Equal(t, "2.0.0-alpha.3", got)
}

func TestResolveCatalogVersionRefusesPrereleaseOnlyHistory(t *testing.T) {
	lister := &fakeLister{versions: []string{"v2.0.0-alpha.1", "v2.0.0-alpha.3"}}

	_, err := resolveCatalogVersion(context.Background(), lister, "ghcr.io/example", DefaultCatalogPath, false)

	var refusal *RefusalError
	require.ErrorAs(t, err, &refusal)
	assert.Contains(t, refusal.Refusal.Action, "--catalog-prerelease")
	assert.Contains(t, refusal.Refusal.Details(), "v2.0.0-alpha.3")

	var conn *publish.ConnectivityError
	assert.False(t, errors.As(err, &conn), "a judged listing is a refusal, never connectivity")
}

func TestResolveCatalogVersionListingFailureIsConnectivity(t *testing.T) {
	lister := &fakeLister{err: errors.New("dial tcp: connection refused")}

	_, err := resolveCatalogVersion(context.Background(), lister, "ghcr.io/example", DefaultCatalogPath, false)

	var conn *publish.ConnectivityError
	require.ErrorAs(t, err, &conn)
	assert.Contains(t, conn.Op, DefaultCatalogPath)
	assert.Contains(t, conn.Op, "ghcr.io/example")

	var refusal *RefusalError
	assert.False(t, errors.As(err, &refusal), "an unjudged listing is never a refusal")
}

func TestResolveCatalogVersionRejectsUnusableRegistry(t *testing.T) {
	_, err := ResolveCatalogVersion(context.Background(), "!!not a registry!!", DefaultCatalogPath, false)

	require.Error(t, err)
}

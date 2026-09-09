package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
)

// The CLI words both advisory facts the kernel reports as rows: a skew the
// warn policy let through (a resolved-versions row marked Newer) and an
// unhandled optional trait. Each row yields exactly one line naming the same
// facts the kernel used to name; dropping either row drops its line.
func TestFormatAdvisories_WordsBothRows(t *testing.T) {
	d := kernel.RenderDiagnostics{
		UnhandledTraits: map[string][]string{
			"web": {"opmodel.dev/catalogs/opm/traits/expose@v1"},
		},
		ResolvedVersions: []kernel.ResolvedVersion{
			{Path: "opmodel.dev/core@v2", ModuleVersion: "v2.0.0-alpha.6", PlatformVersion: "v2.0.0-alpha.6"},
			{Path: "opmodel.dev/catalogs/opm@v4", ModuleVersion: "v4.1.0", PlatformVersion: "v4.0.1", Newer: true},
		},
	}

	got := formatAdvisories(d)
	require.Len(t, got, 2, "one line per advisory row; the non-newer version row is not a warning")
	assert.Equal(t, `version skew on "opmodel.dev/catalogs/opm@v4": module requires v4.1.0, platform carries v4.0.1; rendering against the platform's build`, got[0])
	assert.Equal(t, `component "web": trait "opmodel.dev/catalogs/opm/traits/expose@v1" is not handled by any matched transformer (values will be ignored)`, got[1])

	d.ResolvedVersions = d.ResolvedVersions[:1]
	assert.Len(t, formatAdvisories(d), 1, "dropping the newer row drops the skew line")
	d.UnhandledTraits = nil
	assert.Empty(t, formatAdvisories(d), "dropping the trait row drops its line")
	assert.NotNil(t, formatAdvisories(d), "no advisories is an empty list, not nil")
}

// Unhandled traits print in component order regardless of map iteration, and
// every trait of a component prints.
func TestFormatAdvisories_ComponentOrder(t *testing.T) {
	d := kernel.RenderDiagnostics{
		UnhandledTraits: map[string][]string{
			"worker": {"t@v1"},
			"api":    {"a@v1", "b@v1"},
		},
	}
	got := formatAdvisories(d)
	require.Len(t, got, 3)
	assert.Contains(t, got[0], `component "api": trait "a@v1"`)
	assert.Contains(t, got[1], `component "api": trait "b@v1"`)
	assert.Contains(t, got[2], `component "worker": trait "t@v1"`)
}

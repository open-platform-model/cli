package render

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/library/opm/kernel"
)

func TestFormatSkipped_NoRowsIsEmpty(t *testing.T) {
	assert.Equal(t, []string{}, formatSkipped(nil))
}

func TestFormatSkipped_Trait(t *testing.T) {
	got := formatSkipped([]kernel.SkippedDemand{
		{Component: "db", FQN: "opmodel.dev/catalogs/opm/traits/backup@v1alpha1", Kind: "trait", Alternatives: []string{}},
	})
	assert.Equal(t, []string{
		`component "db": skipped provider-fulfilled trait "opmodel.dev/catalogs/opm/traits/backup@v1alpha1" (no provider on this platform)`,
	}, got)
}

// An omitted component is worded once, naming both of its skipped
// resources, at the position of its first row; its trait row is named on
// the same line, and a trait of a component that rendered keeps its own
// line after it (the kernel emits resource rows before trait rows).
func TestFormatSkipped_OmittedComponentWithTwoResources(t *testing.T) {
	got := formatSkipped([]kernel.SkippedDemand{
		{Component: "archive", FQN: "example.dev/catalogs/k8up/resources/backup-store@v1alpha1", Kind: "resource", ComponentOmitted: true},
		{Component: "archive", FQN: "example.dev/catalogs/k8up/resources/backup-key@v1alpha1", Kind: "resource", ComponentOmitted: true},
		{Component: "db", FQN: "opmodel.dev/catalogs/opm/traits/backup@v1alpha1", Kind: "trait"},
		{Component: "archive", FQN: "opmodel.dev/catalogs/opm/traits/backup@v1alpha1", Kind: "trait", ComponentOmitted: true},
	})
	assert.Equal(t, []string{
		`component "archive" not rendered: provider-fulfilled resources "example.dev/catalogs/k8up/resources/backup-store@v1alpha1", "example.dev/catalogs/k8up/resources/backup-key@v1alpha1" have no provider on this platform; also skipped provider-fulfilled trait(s) "opmodel.dev/catalogs/opm/traits/backup@v1alpha1"`,
		`component "db": skipped provider-fulfilled trait "opmodel.dev/catalogs/opm/traits/backup@v1alpha1" (no provider on this platform)`,
	}, got)
}

func TestFormatSkipped_OmittedComponentOneResource(t *testing.T) {
	got := formatSkipped([]kernel.SkippedDemand{
		{Component: "archive", FQN: "example.dev/catalogs/k8up/resources/backup-store@v1alpha1", Kind: "resource", ComponentOmitted: true},
	})
	assert.Equal(t, []string{
		`component "archive" not rendered: provider-fulfilled resource "example.dev/catalogs/k8up/resources/backup-store@v1alpha1" has no provider on this platform`,
	}, got)
}

func TestFormatSkipped_Alternatives(t *testing.T) {
	got := formatSkipped([]kernel.SkippedDemand{
		{Component: "archive", FQN: "example.dev/r/store@v1alpha1", Kind: "resource", ComponentOmitted: true, Alternatives: []string{"example.dev/r/store@v1alpha2"}},
		{Component: "cache", FQN: "example.dev/r/store@v1alpha1", Kind: "resource", ComponentOmitted: true, Alternatives: []string{"example.dev/r/store@v1alpha2"}},
		{Component: "cache", FQN: "example.dev/r/key@v1alpha1", Kind: "resource", ComponentOmitted: true},
		{Component: "db", FQN: "example.dev/t/backup@v1alpha1", Kind: "trait", Alternatives: []string{"example.dev/t/backup@v1beta1", "example.dev/t/backup@v1"}},
	})
	assert.Equal(t, []string{
		`component "archive" not rendered: provider-fulfilled resource "example.dev/r/store@v1alpha1" has no provider on this platform; implemented at: example.dev/r/store@v1alpha2`,
		`component "cache" not rendered: provider-fulfilled resources "example.dev/r/store@v1alpha1", "example.dev/r/key@v1alpha1" have no provider on this platform; "example.dev/r/store@v1alpha1" implemented at: example.dev/r/store@v1alpha2`,
		`component "db": skipped provider-fulfilled trait "example.dev/t/backup@v1alpha1" (no provider on this platform); implemented at: example.dev/t/backup@v1beta1, example.dev/t/backup@v1`,
	}, got)
}

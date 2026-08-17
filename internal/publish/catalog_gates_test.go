package publish

import (
	"strings"
	"testing"

	cueerrors "cuelang.org/go/cue/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// refusalErrors joins every refusal's CUE error text for containment asserts.
func refusalErrors(p *Plan) string {
	var b strings.Builder
	for _, r := range p.Refusals {
		if r.Err != nil {
			for _, e := range cueerrors.Errors(r.Err) {
				b.WriteString(e.Error())
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

func TestGateMemberFQN_CleanTreePasses(t *testing.T) {
	p := runFixture(t, KindCatalog, memberCatalogFiles())
	require.True(t, p.Go(), refusalHeadlines(p))
	assert.Equal(t, 6, p.CatalogGates.MembersChecked)
	assert.Equal(t, 0, p.CatalogGates.MembersRefused)
	assert.Equal(t, 2, p.CatalogGates.TraitsChecked)
	assert.Equal(t, 0, p.CatalogGates.TraitsRefused)
}

// TestGateMemberFQN_MissingAPIVersionCaughtConcretely pins the measured blind
// spot the gate exists for: a primitive omitting declaredAPIVersion passes
// `cue vet` AND `cue vet -c` (measured in core's identity_package_pins.cue) —
// only concrete evaluation without the incomplete-filter reports it.
func TestGateMemberFQN_MissingAPIVersionCaughtConcretely(t *testing.T) {
	files := memberCatalogFiles()
	files["resources/v1beta1/thing.cue"] = `package v1beta1

#ThingResource: {
	kind: "Resource"
	metadata: {
		modulePath:     "example.com/catalogs/demo/resources/v1beta1"
		name:           "thing"
		catalogVersion: "1.2.0"
		fqn:            "example.com/catalogs/demo/resources/thing@v1beta1"
	}
	spec: {}
}
`
	p := runFixture(t, KindCatalog, files)
	require.False(t, p.Go())
	assert.Equal(t, 1, p.CatalogGates.MembersRefused)
	assert.Contains(t, refusalErrors(p), "declaredAPIVersion")
}

func TestGateMemberFQN_WrongDepthFiling(t *testing.T) {
	// The pre-D49 flat shape: authored modulePath omits the apiVersion
	// segment. The identity-derived filing conflicts with the authored value.
	files := memberCatalogFiles()
	files["resources/v1beta1/thing.cue"] = `package v1beta1

#ThingResource: {
	kind: "Resource"
	metadata: {
		modulePath:     "example.com/catalogs/demo/resources"
		name:           "thing"
		apiVersion:     "v1beta1"
		catalogVersion: "1.2.0"
		fqn:            "example.com/catalogs/demo/resources/thing@v1beta1"
	}
	spec: {}
}
`
	p := runFixture(t, KindCatalog, files)
	require.False(t, p.Go())
	errs := refusalErrors(p)
	assert.Contains(t, errs, "declaredModulePath")
	assert.Contains(t, errs, "conflicting values")
}

func TestGateMemberFQN_WrongKindFiling(t *testing.T) {
	// A trait filed under resources/: the gate takes the FILING directory as
	// its kind, so the authored traits/ paths conflict with the derived
	// resources/ ones — CUE's own diagnostics, not a recomputed comparison.
	files := memberCatalogFiles()
	files["resources/v1beta1/misfiled.cue"] = `package v1beta1

#MisfiledTrait: {
	kind: "Trait"
	metadata: {
		modulePath:     "example.com/catalogs/demo/traits/v1beta1"
		name:           "misfiled"
		apiVersion:     "v1beta1"
		catalogVersion: "1.2.0"
		fqn:            "example.com/catalogs/demo/traits/misfiled@v1beta1"
	}
	optional: bool | *true
	spec: {}
}
`
	p := runFixture(t, KindCatalog, files)
	require.False(t, p.Go())
	errs := refusalErrors(p)
	assert.Contains(t, errs, "conflicting values")
	assert.Contains(t, errs, "resources")
}

func TestGateMemberFQN_StaleCatalogVersion(t *testing.T) {
	files := memberCatalogFiles()
	files["transformers/deploy.cue"] = `package transformers

#DeployTransformer: {
	kind: "ComponentTransformer"
	metadata: {
		modulePath:     "example.com/catalogs/demo/transformers"
		name:           "deploy"
		catalogVersion: "1.1.0"
		fqn:            "example.com/catalogs/demo/transformers/deploy@1.1.0"
	}
}
`
	p := runFixture(t, KindCatalog, files)
	require.False(t, p.Go())
	errs := refusalErrors(p)
	// Both the provenance field and the build-keyed fqn name the stale build
	// — the case 0010 D21 measured passing `cue vet -c` at exit 0.
	assert.Contains(t, errs, "declaredCatalogVersion")
	assert.Contains(t, errs, "1.1.0")
}

func TestGateMemberFQN_TransformerOmitsAPIVersionCleanly(t *testing.T) {
	// Direction one of the conditional optional (D44): a transformer declares
	// no apiVersion and passes — its arm is selected before the absent
	// optional is reached.
	p := runFixture(t, KindCatalog, memberCatalogFiles())
	require.True(t, p.Go(), refusalHeadlines(p))
}

func TestGateTraitOptional_UnstatedPostureRefused(t *testing.T) {
	// Rule 1's measured blind spot: an unstated posture is an incomplete
	// value, invisible to plain vet — the finding exists only concretely.
	files := memberCatalogFiles()
	files["traits/v1beta1/scaling.cue"] = strings.Replace(
		files["traits/v1beta1/scaling.cue"], "optional: bool | *true", "optional: bool", 1)
	p := runFixture(t, KindCatalog, files)
	require.False(t, p.Go())
	assert.Equal(t, 1, p.CatalogGates.TraitsRefused)
	assert.Contains(t, refusalHeadlines(p), "#ScalingTrait")
	assert.Contains(t, refusalErrors(p), "incomplete")
}

func TestGateTraitOptional_PinnedPostureRefused(t *testing.T) {
	files := memberCatalogFiles()
	files["traits/v1beta1/scaling.cue"] = strings.Replace(
		files["traits/v1beta1/scaling.cue"], "optional: bool | *true", "optional: true", 1)
	p := runFixture(t, KindCatalog, files)
	require.False(t, p.Go())
	assert.Equal(t, 1, p.CatalogGates.TraitsRefused)
	assert.Contains(t, refusalErrors(p), "_overridable")
}

func TestGateTraitOptional_AlphaTraitsIncluded(t *testing.T) {
	// The D34 alpha carve-out is compat-only: a v1alpha1 trait pinning its
	// posture is refused like any other. Asserted, not assumed.
	files := memberCatalogFiles()
	files["traits/v1alpha1/experimental.cue"] = strings.Replace(
		files["traits/v1alpha1/experimental.cue"], "optional: bool | *false", "optional: false", 1)
	p := runFixture(t, KindCatalog, files)
	require.False(t, p.Go())
	assert.Equal(t, 1, p.CatalogGates.TraitsRefused)
	assert.Contains(t, refusalHeadlines(p), "#ExperimentalTrait")
}

func TestRun_ModuleKindSkipsCatalogGates(t *testing.T) {
	p := runFixture(t, KindModule, moduleFiles())
	require.True(t, p.Go(), refusalHeadlines(p))
	assert.Zero(t, p.CatalogGates.MembersChecked)
	assert.NotContains(t, p.Render(), "member gate")
}

func TestRender_CatalogGateCounts(t *testing.T) {
	p := runFixture(t, KindCatalog, memberCatalogFiles())
	out := p.Render()
	assert.Contains(t, out, "member gate")
	assert.Contains(t, out, "6 members checked, 0 refused")
	assert.Contains(t, out, "posture gate")
	assert.Contains(t, out, "2 traits checked, 0 refused")
}

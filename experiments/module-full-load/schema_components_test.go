package modulefullload

// ---------------------------------------------------------------------------
// Decisions 4 + 5: Schema-level component extraction
//
// core.ExtractComponents(v cue.Value) is called by module.Load() on the
// #components value at schema level — before any user values are applied.
// At this point spec fields like spec.image are still `string` (a type
// constraint), not "nginx:latest" (a concrete value).
//
// The design must handle this gracefully:
//   - Component metadata (name, labels) IS concrete — readable for matching
//   - Component resources/traits keys ARE iterable — readable for matching
//   - Component spec fields are NOT concrete — Validate(Concrete(true)) fails
//   - Component.Validate() (structural) PASSES at schema level
//   - Component.IsConcrete() RETURNS FALSE at schema level
//
// These tests prove all of the above, establishing the invariant that
// schema-level extraction is valid and useful (for transformer matching),
// and that IsConcrete() correctly gates concrete-only operations.
// ---------------------------------------------------------------------------

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simpleComponent is a minimal Go representation of a schema-level component,
// simulating what core.Component will look like after the design change.
// (We don't import internal/core — this experiment is detached.)
type simpleComponent struct {
	Name      string
	Labels    map[string]string
	Resources map[string]cue.Value // FQN -> resource cue.Value
	Traits    map[string]cue.Value // FQN -> trait cue.Value (may be empty)
	Value     cue.Value            // full component cue.Value
}

// validate simulates Component.Validate() from Decision 4:
// checks structural requirements without concreteness.
func (c *simpleComponent) validate() error {
	if c.Name == "" {
		return assert.AnError // stand-in
	}
	if len(c.Resources) == 0 {
		return assert.AnError
	}
	if !c.Value.Exists() {
		return assert.AnError
	}
	return nil
}

// isConcrete simulates Component.IsConcrete() from Decision 4.
func (c *simpleComponent) isConcrete() bool {
	return c.Value.Validate(cue.Concrete(true)) == nil
}

// extractComponents simulates core.ExtractComponents(v cue.Value).
// Iterates the #components struct and populates simpleComponent for each entry.
func extractComponents(t *testing.T, componentsVal cue.Value) map[string]*simpleComponent {
	t.Helper()
	result := map[string]*simpleComponent{}

	iter, err := componentsVal.Fields(cue.Hidden(true))
	require.NoError(t, err, "should be able to iterate #components")

	for iter.Next() {
		name := iter.Label()
		compVal := iter.Value()

		comp := &simpleComponent{
			Name:      name,
			Labels:    map[string]string{},
			Resources: map[string]cue.Value{},
			Traits:    map[string]cue.Value{},
			Value:     compVal,
		}

		// Extract metadata.name and metadata.labels.
		metaName, err := compVal.LookupPath(cue.ParsePath("metadata.name")).String()
		if err == nil {
			comp.Name = metaName
		}

		labelsVal := compVal.LookupPath(cue.ParsePath("metadata.labels"))
		if labelsVal.Exists() {
			lIter, err := labelsVal.Fields()
			if err == nil {
				for lIter.Next() {
					if v, err := lIter.Value().String(); err == nil {
						comp.Labels[lIter.Label()] = v
					}
				}
			}
		}

		// Extract #resources keys.
		resourcesVal := compVal.LookupPath(cue.MakePath(cue.Def("resources")))
		if resourcesVal.Exists() && resourcesVal.Err() == nil {
			rIter, err := resourcesVal.Fields()
			if err == nil {
				for rIter.Next() {
					comp.Resources[rIter.Label()] = rIter.Value()
				}
			}
		}

		// Extract #traits keys (optional).
		traitsVal := compVal.LookupPath(cue.MakePath(cue.Def("traits")))
		if traitsVal.Exists() && traitsVal.Err() == nil {
			tIter, err := traitsVal.Fields()
			if err == nil {
				for tIter.Next() {
					comp.Traits[tIter.Label()] = tIter.Value()
				}
			}
		}

		result[name] = comp
	}

	return result
}

// TestSchemaExtract_ComponentsExist proves the #components definition is
// accessible on the evaluated base value.
func TestSchemaExtract_ComponentsExist(t *testing.T) {
	_, val := buildBaseValue(t)
	componentsVal := val.LookupPath(cue.MakePath(cue.Def("components")))
	assert.True(t, componentsVal.Exists(), "#components should exist in evaluated value")
	assert.NoError(t, componentsVal.Err())
}

// TestSchemaExtract_IterateComponents proves that iterating #components yields
// the expected component keys.
func TestSchemaExtract_IterateComponents(t *testing.T) {
	_, val := buildBaseValue(t)
	componentsVal := val.LookupPath(cue.MakePath(cue.Def("components")))

	comps := extractComponents(t, componentsVal)
	assert.Len(t, comps, 2, "should find 2 components")
	assert.Contains(t, comps, "web")
	assert.Contains(t, comps, "worker")
}

// TestSchemaExtract_ComponentMetadataReadable proves that component metadata
// (name, labels) is concrete and readable at schema level. These are static
// string literals in the module — they don't depend on #config values.
func TestSchemaExtract_ComponentMetadataReadable(t *testing.T) {
	_, val := buildBaseValue(t)
	componentsVal := val.LookupPath(cue.MakePath(cue.Def("components")))
	comps := extractComponents(t, componentsVal)

	web, ok := comps["web"]
	require.True(t, ok)
	assert.Equal(t, "web", web.Name)
	assert.Equal(t, "stateless", web.Labels["workload-type"])

	worker, ok := comps["worker"]
	require.True(t, ok)
	assert.Equal(t, "worker", worker.Name)
	assert.Equal(t, "worker", worker.Labels["workload-type"])
}

// TestSchemaExtract_ResourcesIterable proves that #resources keys are
// iterable at schema level. This is used for transformer matching by FQN.
func TestSchemaExtract_ResourcesIterable(t *testing.T) {
	_, val := buildBaseValue(t)
	componentsVal := val.LookupPath(cue.MakePath(cue.Def("components")))
	comps := extractComponents(t, componentsVal)

	web := comps["web"]
	require.NotEmpty(t, web.Resources, "web should have at least one resource")

	found := false
	for fqn := range web.Resources {
		if strings.Contains(fqn, "Container") {
			found = true
			break
		}
	}
	assert.True(t, found, "web should have a Container resource")

	worker := comps["worker"]
	require.NotEmpty(t, worker.Resources, "worker should have at least one resource")
}

// TestSchemaExtract_SpecHasSchemaDefaults proves that component spec fields
// resolve to their schema defaults at schema level — because all #config fields
// carry defaults (e.g. image: string | *"nginx:latest"). CUE evaluates the
// disjunction to the default, making spec fields concrete before any user
// values are applied.
//
// This is the key design insight: a module with complete defaults is already
// "deployable" at schema level (using those defaults). FillPath overrides them.
func TestSchemaExtract_SpecHasSchemaDefaults(t *testing.T) {
	_, val := buildBaseValue(t)
	webSpec := val.LookupPath(cue.ParsePath("#components.web.spec"))
	require.True(t, webSpec.Exists())

	// spec.image resolves to the #config.image default "nginx:latest".
	image, err := webSpec.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, err, "spec.image should be concrete (schema default) at schema level")
	assert.Equal(t, "nginx:latest", image, "spec.image schema default should be nginx:latest")

	// spec.replicas resolves to the #config.replicas default 1.
	replicas, err := webSpec.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err, "spec.replicas should be concrete (schema default) at schema level")
	assert.Equal(t, int64(1), replicas, "spec.replicas schema default should be 1")

	// spec.port resolves to the #config.port default 8080.
	port, err := webSpec.LookupPath(cue.ParsePath("port")).Int64()
	require.NoError(t, err, "spec.port should be concrete (schema default) at schema level")
	assert.Equal(t, int64(8080), port, "spec.port schema default should be 8080")
}

// TestSchemaExtract_Validate_Structural proves that Component.Validate()
// (structural check) passes at schema level. Structural validation checks
// that Name, Resources, and Value are present — not that values are concrete.
func TestSchemaExtract_Validate_Structural(t *testing.T) {
	_, val := buildBaseValue(t)
	componentsVal := val.LookupPath(cue.MakePath(cue.Def("components")))
	comps := extractComponents(t, componentsVal)

	for name, comp := range comps {
		err := comp.validate()
		assert.NoError(t, err, "component %q should pass structural validation at schema level", name)
	}
}

// TestSchemaExtract_IsConcreteWithDefaults proves that Component.IsConcrete()
// returns true at schema level when all #config fields carry defaults. Because
// CUE resolves disjunctions to their defaults at evaluation time, a fully-defaulted
// module is already concrete before any user values are applied.
//
// FillPath() still matters: it lets user values override those defaults.
// But a module that provides defaults for everything is immediately deployable.
func TestSchemaExtract_IsConcreteWithDefaults(t *testing.T) {
	_, val := buildBaseValue(t)
	componentsVal := val.LookupPath(cue.MakePath(cue.Def("components")))
	comps := extractComponents(t, componentsVal)

	require.NotEmpty(t, comps)
	for name, comp := range comps {
		assert.True(t, comp.isConcrete(),
			"component %q SHOULD be concrete at schema level (all #config fields have defaults)", name)
	}
}

// TestSchemaExtract_TraitsOptional proves that a component without #traits
// (worker) passes validation and extraction without error. Traits are optional
// in the design — only resources are required.
func TestSchemaExtract_TraitsOptional(t *testing.T) {
	_, val := buildBaseValue(t)
	componentsVal := val.LookupPath(cue.MakePath(cue.Def("components")))
	comps := extractComponents(t, componentsVal)

	worker := comps["worker"]
	require.NotNil(t, worker)

	// worker has no #traits — Traits map should be empty.
	assert.Empty(t, worker.Traits, "worker should have no traits")

	// But validation still passes — traits are optional.
	assert.NoError(t, worker.validate(), "worker should pass validation without traits")
}

// TestSchemaExtract_WebHasTraits proves that a component with #traits (web)
// has them populated after extraction.
func TestSchemaExtract_WebHasTraits(t *testing.T) {
	_, val := buildBaseValue(t)
	componentsVal := val.LookupPath(cue.MakePath(cue.Def("components")))
	comps := extractComponents(t, componentsVal)

	web := comps["web"]
	require.NotNil(t, web)
	assert.NotEmpty(t, web.Traits, "web should have at least one trait")
}

package publish

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memberCatalogFiles is catalogFiles grown a full member tree: one beta and
// one GA resource, a beta and an alpha trait, a blueprint, a flat transformer
// — plus the shapes the walk must NOT return: fragments and a component
// wrapper beside members, and a member-shaped definition in schemas/ (the
// directory a value walk would have been dragged into via references).
func memberCatalogFiles() map[string]string {
	files := catalogFiles()
	files["resources/v1beta1/thing.cue"] = `package v1beta1

#ThingResource: {
	kind: "Resource"
	metadata: {
		modulePath:     "example.com/catalogs/demo/resources/v1beta1"
		name:           "thing"
		apiVersion:     "v1beta1"
		catalogVersion: "1.2.0"
		fqn:            "example.com/catalogs/demo/resources/thing@v1beta1"
	}
	spec: {
		size: int
		note?: string
	}
}

// A fragment beside the member: no kind discriminator, never enumerated.
#ThingSchema: {
	size: int
}

// A component wrapper: kinded, but not a member kind.
#Thing: {
	kind: "Component"
	#resources: (#ThingResource.metadata.fqn): #ThingResource
}
`
	files["resources/v1/raw.cue"] = `package v1

#RawResource: {
	kind: "Resource"
	metadata: {
		modulePath:     "example.com/catalogs/demo/resources/v1"
		name:           "raw"
		apiVersion:     "v1"
		catalogVersion: "1.2.0"
		fqn:            "example.com/catalogs/demo/resources/raw@v1"
	}
	spec: {
		image: string
	}
}
`
	files["traits/v1beta1/scaling.cue"] = `package v1beta1

#ScalingTrait: {
	kind: "Trait"
	metadata: {
		modulePath:     "example.com/catalogs/demo/traits/v1beta1"
		name:           "scaling"
		apiVersion:     "v1beta1"
		catalogVersion: "1.2.0"
		fqn:            "example.com/catalogs/demo/traits/scaling@v1beta1"
	}
	optional: bool | *true
	spec: {
		replicas: int
	}
}
`
	files["traits/v1alpha1/experimental.cue"] = `package v1alpha1

#ExperimentalTrait: {
	kind: "Trait"
	metadata: {
		modulePath:     "example.com/catalogs/demo/traits/v1alpha1"
		name:           "experimental"
		apiVersion:     "v1alpha1"
		catalogVersion: "1.2.0"
		fqn:            "example.com/catalogs/demo/traits/experimental@v1alpha1"
	}
	optional: bool | *false
	spec: {
		knob: string
	}
}
`
	files["blueprints/v1beta1/workload.cue"] = `package v1beta1

#WorkloadBlueprint: {
	kind: "Blueprint"
	metadata: {
		modulePath:     "example.com/catalogs/demo/blueprints/v1beta1"
		name:           "workload"
		apiVersion:     "v1beta1"
		catalogVersion: "1.2.0"
		fqn:            "example.com/catalogs/demo/blueprints/workload@v1beta1"
	}
	spec: {
		image: string
	}
}
`
	files["transformers/deploy.cue"] = `package transformers

#DeployTransformer: {
	kind: "ComponentTransformer"
	metadata: {
		modulePath:     "example.com/catalogs/demo/transformers"
		name:           "deploy"
		catalogVersion: "1.2.0"
		fqn:            "example.com/catalogs/demo/transformers/deploy@1.2.0"
	}
}
`
	// A member-shaped definition outside the kind directories: the walk never
	// visits schemas/, so it is excluded structurally — the same mechanism
	// that keeps referenced foreign primitives out (0010 D17).
	files["schemas/foreign.cue"] = `package schemas

#ForeignResource: {
	kind: "Resource"
	metadata: {
		modulePath:     "example.com/catalogs/other/resources/v1beta1"
		name:           "foreign"
		apiVersion:     "v1beta1"
		catalogVersion: "9.9.9"
		fqn:            "example.com/catalogs/other/resources/foreign@v1beta1"
	}
	spec: {}
}
`
	return files
}

func enumerateFixture(t *testing.T, files map[string]string) ([]Member, []Refusal) {
	t.Helper()
	dir := writeTree(t, files)
	opts := baseOptions(t, dir)
	return enumerateMembers(opts, dir)
}

func TestEnumerateMembers_CountsPerKind(t *testing.T) {
	members, refusals := enumerateFixture(t, memberCatalogFiles())
	require.Empty(t, refusals)

	byKind := map[string][]string{}
	for _, m := range members {
		byKind[m.Kind] = append(byKind[m.Kind], m.Name)
	}
	assert.ElementsMatch(t, []string{"thing", "raw"}, byKind["resources"])
	assert.ElementsMatch(t, []string{"scaling", "experimental"}, byKind["traits"])
	assert.ElementsMatch(t, []string{"workload"}, byKind["blueprints"])
	assert.ElementsMatch(t, []string{"deploy"}, byKind["transformers"])
	assert.Len(t, members, 6)
}

func TestEnumerateMembers_FragmentsAndWrappersExcluded(t *testing.T) {
	members, refusals := enumerateFixture(t, memberCatalogFiles())
	require.Empty(t, refusals)
	for _, m := range members {
		assert.NotEqual(t, "#ThingSchema", m.DefName, "fragment beside a member must not be enumerated")
		assert.NotEqual(t, "#Thing", m.DefName, "component wrapper must not be enumerated")
	}
}

func TestEnumerateMembers_OutsideKindDirsNeverVisited(t *testing.T) {
	members, refusals := enumerateFixture(t, memberCatalogFiles())
	require.Empty(t, refusals)
	for _, m := range members {
		assert.NotEqual(t, "foreign", m.Name, "schemas/ is not a member package; the walk must not reach it")
	}
}

func TestEnumerateMembers_ModelFields(t *testing.T) {
	members, refusals := enumerateFixture(t, memberCatalogFiles())
	require.Empty(t, refusals)

	find := func(name string) Member {
		for _, m := range members {
			if m.Name == name {
				return m
			}
		}
		t.Fatalf("member %q not enumerated", name)
		return Member{}
	}

	thing := find("thing")
	assert.Equal(t, "resources", thing.Kind)
	assert.Equal(t, "Resource", thing.Discriminator)
	assert.Equal(t, "#ThingResource", thing.DefName)
	assert.Equal(t, "v1beta1", thing.APIVersion)
	assert.Equal(t, "resources/v1beta1", thing.PkgPath)
	assert.True(t, thing.Value.Exists())

	deploy := find("deploy")
	assert.Equal(t, "transformers", deploy.Kind)
	assert.Equal(t, "ComponentTransformer", deploy.Discriminator)
	assert.Equal(t, "", deploy.APIVersion, "a transformer carries no apiVersion (D44)")
	assert.Equal(t, "transformers", deploy.PkgPath)
}

func TestEnumerateMembers_FlatFiledContractMemberStillReached(t *testing.T) {
	// A contract member filed flat at its kind prefix (the pre-D49 shape)
	// must be enumerated so the FQN gate can refuse it — invisible would mean
	// unrefusable.
	files := catalogFiles()
	files["resources/flat.cue"] = `package resources

#FlatResource: {
	kind: "Resource"
	metadata: {
		modulePath:     "example.com/catalogs/demo/resources"
		name:           "flat"
		apiVersion:     "v1beta1"
		catalogVersion: "1.2.0"
		fqn:            "example.com/catalogs/demo/resources/flat@v1beta1"
	}
	spec: {}
}
`
	members, refusals := enumerateFixture(t, files)
	require.Empty(t, refusals)
	require.Len(t, members, 1)
	assert.Equal(t, "flat", members[0].Name)
	assert.Equal(t, "resources", members[0].PkgPath)
}

func TestEnumerateMembers_BrokenPackageIsARefusal(t *testing.T) {
	files := memberCatalogFiles()
	files["traits/v1beta1/broken.cue"] = `package v1beta1

#Broken: { kind: "Trait", metadata: name: 42 & "x" }
`
	_, refusals := enumerateFixture(t, files)
	require.NotEmpty(t, refusals)
	assert.Contains(t, refusals[0].Headline, "traits/v1beta1")
}

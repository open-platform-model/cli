package operator

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/semver"
)

// legacyRelease is one release of testdata/legacy-manifests.json, as
// hack/operator-legacy writes it.
type legacyRelease struct {
	Tag     string `json:"tag"`
	Objects []struct {
		Group     string            `json:"group"`
		Kind      string            `json:"kind"`
		Namespace string            `json:"namespace"`
		Name      string            `json:"name"`
		Labels    map[string]string `json:"labels"`
		Selector  map[string]string `json:"selector"`
	} `json:"objects"`
}

func readLegacyManifests(t *testing.T) []legacyRelease {
	t.Helper()
	data, err := os.ReadFile("testdata/legacy-manifests.json")
	require.NoError(t, err)
	var rels []legacyRelease
	require.NoError(t, json.Unmarshal(data, &rels))
	require.NotEmpty(t, rels)
	return rels
}

// legacyUnion folds the recorded releases from FirstLegacyRelease on into
// the proof list's shape: one entry per object, with the first and last
// release that shipped it. It fails when an object's labels or selector
// differ between releases, since the proof keeps one set per object.
func legacyUnion(t *testing.T, rels []legacyRelease) map[string]LegacyObject {
	t.Helper()
	union := map[string]LegacyObject{}
	for _, rel := range rels {
		if semver.Compare(rel.Tag, FirstLegacyRelease) < 0 {
			continue
		}
		for _, o := range rel.Objects {
			key := o.Group + "/" + o.Kind + "/" + o.Namespace + "/" + o.Name
			got, seen := union[key]
			if !seen {
				union[key] = LegacyObject{
					Group: o.Group, Kind: o.Kind, Namespace: o.Namespace, Name: o.Name,
					Labels: o.Labels, Selector: o.Selector, Releases: rel.Tag + ".." + rel.Tag,
				}
				continue
			}
			assert.Equal(t, got.Labels, o.Labels, "%s keeps one label set across releases (%s)", key, rel.Tag)
			assert.Equal(t, got.Selector, o.Selector, "%s keeps one selector across releases (%s)", key, rel.Tag)
			first, _, _ := strings.Cut(got.Releases, "..")
			got.Releases = first + ".." + rel.Tag
			union[key] = got
		}
	}
	return union
}

// LegacyObjects is exactly the union of the recorded operator manifests.
func TestLegacyObjects_EqualTheRecordedManifests(t *testing.T) {
	rels := readLegacyManifests(t)
	for i := 1; i < len(rels); i++ {
		require.Negative(t, semver.Compare(rels[i-1].Tag, rels[i].Tag), "releases are recorded in order")
	}
	union := legacyUnion(t, rels)

	listed := map[string]LegacyObject{}
	for _, o := range LegacyObjects {
		key := o.Group + "/" + o.Kind + "/" + o.Namespace + "/" + o.Name
		_, dup := listed[key]
		require.False(t, dup, "%s is listed once", key)
		listed[key] = o
	}
	assert.Equal(t, union, listed)
}

// Every recorded release is an operator release; none is an operator
// module release.
func TestLegacyManifests_OnlyOperatorReleases(t *testing.T) {
	for _, rel := range readLegacyManifests(t) {
		assert.True(t, semver.IsValid(rel.Tag), rel.Tag)
		assert.False(t, strings.HasPrefix(rel.Tag, "opm_operator"), rel.Tag)
	}
}

// The releases before FirstLegacyRelease install another operator, under
// other names in another namespace: leaving them out drops nothing the
// module renders.
func TestLegacyManifests_EarlierReleasesAreAnotherOperator(t *testing.T) {
	for _, rel := range readLegacyManifests(t) {
		if semver.Compare(rel.Tag, FirstLegacyRelease) >= 0 {
			continue
		}
		for _, o := range rel.Objects {
			assert.NotEqual(t, OperatorNamespace, o.Namespace, "%s %s", rel.Tag, o.Name)
			assert.False(t, strings.HasPrefix(o.Name, "opm-operator"), "%s %s", rel.Tag, o.Name)
		}
	}
}

// The only bindings on the list are the three the module supersedes, and
// the controller Deployment carries the one earlier selector.
func TestLegacyObjects_BindingsAndDeployment(t *testing.T) {
	var bindings []string
	for _, o := range LegacyObjects {
		if o.Kind == "ClusterRoleBinding" || o.Kind == "RoleBinding" {
			bindings = append(bindings, o.Kind+"/"+o.Name)
		}
	}
	superseded := make([]string, 0, len(SupersededBindings))
	for _, o := range SupersededBindings {
		superseded = append(superseded, o.Kind+"/"+o.Name)
	}
	assert.ElementsMatch(t, bindings, superseded)

	assert.Equal(t, map[string]string{"app.kubernetes.io/name": "opm-operator", "control-plane": "controller-manager"}, legacyDeployment.Selector)
	assert.Equal(t, OperatorNamespace, legacyDeployment.Namespace)
}

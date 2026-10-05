package apply

import (
	"testing"

	"github.com/stretchr/testify/assert"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
)

// staleEntry builds an inventory entry in namespace "ns" at API version v1.
func staleEntry(group, kind, name, component string) k8sinventory.Entry {
	return k8sinventory.Entry{Group: group, Kind: kind, Namespace: "ns", Name: name, Version: "v1", Component: component}
}

// The apply's stale set keeps every prune decision the CLI made before it
// adopted the library's component-blind stale set: the cases of its earlier
// stale-set rule and of the component-rename filter that followed it, each
// with the expected set it asserted (0012:D7:R1).
func TestComputeStaleInventorySet(t *testing.T) {
	cases := []struct {
		name     string
		previous []k8sinventory.Entry
		current  []k8sinventory.Entry
		want     []string // names of the stale entries, in previous order
	}{
		{
			name: "resource removed",
			previous: []k8sinventory.Entry{
				staleEntry("apps", "Deployment", "app-a", "web"),
				staleEntry("apps", "Deployment", "app-b", "web"),
				staleEntry("", "Service", "svc-a", "web"),
			},
			current: []k8sinventory.Entry{
				staleEntry("apps", "Deployment", "app-a", "web"),
				staleEntry("", "Service", "svc-a", "web"),
			},
			want: []string{"app-b"},
		},
		{
			name:     "resource renamed",
			previous: []k8sinventory.Entry{staleEntry("", "Service", "old-name", "web")},
			current:  []k8sinventory.Entry{staleEntry("", "Service", "new-name", "web")},
			want:     []string{"old-name"},
		},
		{
			name:    "first apply",
			current: []k8sinventory.Entry{staleEntry("apps", "Deployment", "app", "web")},
		},
		{
			name:     "empty previous",
			previous: []k8sinventory.Entry{},
			current:  []k8sinventory.Entry{staleEntry("apps", "Deployment", "app", "web")},
		},
		{
			name: "idempotent re-apply",
			previous: []k8sinventory.Entry{
				staleEntry("apps", "Deployment", "app", "web"),
				staleEntry("", "Service", "svc", "web"),
			},
			current: []k8sinventory.Entry{
				staleEntry("apps", "Deployment", "app", "web"),
				staleEntry("", "Service", "svc", "web"),
			},
		},
		{
			name:     "API version change",
			previous: []k8sinventory.Entry{{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}},
			current:  []k8sinventory.Entry{{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v2", Component: "web"}},
		},
		{
			name:     "component rename kept",
			previous: []k8sinventory.Entry{staleEntry("apps", "Deployment", "my-app", "web")},
			current:  []k8sinventory.Entry{staleEntry("apps", "Deployment", "my-app", "frontend")},
		},
		{
			name:     "genuine removal under the same component",
			previous: []k8sinventory.Entry{staleEntry("apps", "Deployment", "old-app", "web")},
			current:  []k8sinventory.Entry{staleEntry("apps", "Deployment", "new-app", "web")},
			want:     []string{"old-app"},
		},
		{
			name: "mixed rename and removal",
			previous: []k8sinventory.Entry{
				staleEntry("apps", "Deployment", "renamed-app", "old-comp"),
				staleEntry("apps", "Deployment", "removed-app", "web"),
			},
			current: []k8sinventory.Entry{
				staleEntry("apps", "Deployment", "renamed-app", "new-comp"),
				staleEntry("", "Service", "some-svc", "web"),
			},
			want: []string{"removed-app"},
		},
		{
			// The rename filter's same-component input (stale and current
			// both [my-app/web]) cannot come from a stale set; as a
			// (previous, current) pair it is an idempotent re-apply.
			name:     "same component, same object",
			previous: []k8sinventory.Entry{staleEntry("apps", "Deployment", "my-app", "web")},
			current:  []k8sinventory.Entry{staleEntry("apps", "Deployment", "my-app", "web")},
		},
		{
			name:     "component and API version change together",
			previous: []k8sinventory.Entry{{Group: "autoscaling", Kind: "HorizontalPodAutoscaler", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}},
			current:  []k8sinventory.Entry{{Group: "autoscaling", Kind: "HorizontalPodAutoscaler", Namespace: "ns", Name: "app", Version: "v2", Component: "frontend"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stale := ComputeStaleInventorySet(tc.previous, tc.current)
			assert.NotNil(t, stale)
			names := make([]string, 0, len(stale))
			for _, e := range stale {
				names = append(names, e.Name)
			}
			if tc.want == nil {
				tc.want = []string{}
			}
			assert.Equal(t, tc.want, names)
		})
	}
}

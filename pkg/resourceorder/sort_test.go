package resourceorder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type item struct {
	gvk  schema.GroupVersionKind
	name string
}

func itemGVK(i item) schema.GroupVersionKind { return i.gvk }

var (
	gvkDeployment = schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	gvkCRD        = schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}
	gvkConfigMap  = schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}
)

func names(items []item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.name
	}
	return out
}

func TestSort(t *testing.T) {
	tests := []struct {
		name string
		in   []item
		dir  Direction
		want []string
	}{
		{
			name: "ascending for apply",
			in:   []item{{gvkDeployment, "deploy"}, {gvkCRD, "crd"}, {gvkConfigMap, "cm"}},
			dir:  Ascending,
			want: []string{"crd", "cm", "deploy"},
		},
		{
			name: "descending for delete",
			in:   []item{{gvkDeployment, "deploy"}, {gvkCRD, "crd"}, {gvkConfigMap, "cm"}},
			dir:  Descending,
			want: []string{"deploy", "cm", "crd"},
		},
		{
			name: "equal weights keep input order ascending",
			in:   []item{{gvkConfigMap, "b"}, {gvkConfigMap, "a"}},
			dir:  Ascending,
			want: []string{"b", "a"},
		},
		{
			name: "equal weights keep input order descending",
			in:   []item{{gvkConfigMap, "b"}, {gvkDeployment, "d"}, {gvkConfigMap, "a"}},
			dir:  Descending,
			want: []string{"d", "b", "a"},
		},
		{
			name: "empty",
			in:   nil,
			dir:  Ascending,
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Sort(tt.in, itemGVK, tt.dir)
			assert.Equal(t, tt.want, names(tt.in))
		})
	}
}

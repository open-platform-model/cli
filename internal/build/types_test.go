package build

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRenderOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    RenderOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid options with module path",
			opts: RenderOptions{
				ModulePath: "./my-module",
			},
			wantErr: false,
		},
		{
			name: "valid options with all fields",
			opts: RenderOptions{
				ModulePath: "./my-module",
				Values:     []string{"values.cue", "prod.cue"},
				Name:       "my-release",
				Namespace:  "production",
				Provider:   "kubernetes",
				Strict:     true,
				Registry:   "https://registry.example.com",
			},
			wantErr: false,
		},
		{
			name:    "missing module path",
			opts:    RenderOptions{},
			wantErr: true,
			errMsg:  "ModulePath is required",
		},
		{
			name: "empty module path",
			opts: RenderOptions{
				ModulePath: "",
				Namespace:  "default",
			},
			wantErr: true,
			errMsg:  "ModulePath is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRenderResult_HasErrors(t *testing.T) {
	tests := []struct {
		name   string
		result *RenderResult
		want   bool
	}{
		{
			name:   "no errors",
			result: &RenderResult{},
			want:   false,
		},
		{
			name: "empty errors slice",
			result: &RenderResult{
				Errors: []error{},
			},
			want: false,
		},
		{
			name: "with errors",
			result: &RenderResult{
				Errors: []error{errors.New("test error")},
			},
			want: true,
		},
		{
			name: "with multiple errors",
			result: &RenderResult{
				Errors: []error{
					errors.New("error 1"),
					errors.New("error 2"),
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.result.HasErrors())
		})
	}
}

func TestRenderResult_HasWarnings(t *testing.T) {
	tests := []struct {
		name   string
		result *RenderResult
		want   bool
	}{
		{
			name:   "no warnings",
			result: &RenderResult{},
			want:   false,
		},
		{
			name: "empty warnings slice",
			result: &RenderResult{
				Warnings: []string{},
			},
			want: false,
		},
		{
			name: "with warnings",
			result: &RenderResult{
				Warnings: []string{"deprecated transformer"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.result.HasWarnings())
		})
	}
}

func TestRenderResult_ResourceCount(t *testing.T) {
	tests := []struct {
		name   string
		result *RenderResult
		want   int
	}{
		{
			name:   "no resources",
			result: &RenderResult{},
			want:   0,
		},
		{
			name: "empty resources slice",
			result: &RenderResult{
				Resources: []*Resource{},
			},
			want: 0,
		},
		{
			name: "with resources",
			result: &RenderResult{
				Resources: []*Resource{
					{Component: "web"},
					{Component: "api"},
					{Component: "db"},
				},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.result.ResourceCount())
		})
	}
}

func TestResource_Accessors(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "web-server",
				"namespace": "production",
				"labels": map[string]interface{}{
					"app":     "web",
					"version": "v1",
				},
			},
		},
	}

	resource := &Resource{
		Object:      obj,
		Component:   "web",
		Transformer: "opmodel.dev/transformers/kubernetes@v0#DeploymentTransformer",
	}

	t.Run("GVK", func(t *testing.T) {
		gvk := resource.GVK()
		assert.Equal(t, "apps", gvk.Group)
		assert.Equal(t, "v1", gvk.Version)
		assert.Equal(t, "Deployment", gvk.Kind)
	})

	t.Run("Kind", func(t *testing.T) {
		assert.Equal(t, "Deployment", resource.Kind())
	})

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "web-server", resource.Name())
	})

	t.Run("Namespace", func(t *testing.T) {
		assert.Equal(t, "production", resource.Namespace())
	})

	t.Run("Labels", func(t *testing.T) {
		labels := resource.Labels()
		assert.Equal(t, "web", labels["app"])
		assert.Equal(t, "v1", labels["version"])
	})
}

func TestResource_ClusterScoped(t *testing.T) {
	// Test cluster-scoped resource (no namespace)
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRole",
			"metadata": map[string]interface{}{
				"name": "admin-role",
			},
		},
	}

	resource := &Resource{
		Object:      obj,
		Component:   "rbac",
		Transformer: "opmodel.dev/transformers/kubernetes@v0#ClusterRoleTransformer",
	}

	assert.Equal(t, "", resource.Namespace())
	assert.Equal(t, "admin-role", resource.Name())
	assert.Equal(t, "ClusterRole", resource.Kind())
}

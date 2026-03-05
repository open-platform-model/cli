package moduleimport

import (
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFlattenedModuleImport tests whether a flattened module
// (core.#Module embedded at package root) can be loaded and used.
func TestFlattenedModuleImport(t *testing.T) {
	tests := []struct {
		name        string
		modulePath  string
		expectError bool
		checks      func(t *testing.T, v cue.Value)
	}{
		{
			name:       "simple_module_without_values",
			modulePath: "testdata/simple_module",
			checks: func(t *testing.T, v cue.Value) {
				t.Log("Q1: Can we load a flattened module?")
				require.NoError(t, v.Err(), "module should load without error")

				// Check that the value has Module fields
				apiVersion := v.LookupPath(cue.ParsePath("apiVersion"))
				assert.True(t, apiVersion.Exists(), "apiVersion should exist")
				assert.Equal(t, "opmodel.dev/core/v1alpha1", mustString(t, apiVersion))

				kind := v.LookupPath(cue.ParsePath("kind"))
				assert.True(t, kind.Exists(), "kind should exist")
				assert.Equal(t, "Module", mustString(t, kind))

				metadata := v.LookupPath(cue.ParsePath("metadata"))
				assert.True(t, metadata.Exists(), "metadata should exist")

				name := metadata.LookupPath(cue.ParsePath("name"))
				assert.Equal(t, "simple", mustString(t, name))

				t.Log("Q2: Are hidden definitions (#config, #components) accessible?")
				config := v.LookupPath(cue.ParsePath("#config"))
				assert.True(t, config.Exists(), "#config should exist and be accessible")

				components := v.LookupPath(cue.ParsePath("#components"))
				assert.True(t, components.Exists(), "#components should exist and be accessible")

				web := components.LookupPath(cue.ParsePath("web"))
				assert.True(t, web.Exists(), "web component should exist")

				t.Log("✓ Q1 & Q2: Flattened module loads and hidden fields are accessible")
			},
		},
		{
			name:       "module_with_values_field",
			modulePath: "testdata/module_with_values",
			checks: func(t *testing.T, v cue.Value) {
				t.Log("Q3: Does a module with extra 'values' field still load?")
				require.NoError(t, v.Err(), "module should load")

				// Check Module fields exist
				apiVersion := v.LookupPath(cue.ParsePath("apiVersion"))
				assert.True(t, apiVersion.Exists(), "apiVersion should exist")

				// Check the extra values field exists
				values := v.LookupPath(cue.ParsePath("values"))
				assert.True(t, values.Exists(), "values field should exist at package root")

				replicas := values.LookupPath(cue.ParsePath("replicas"))
				assert.Equal(t, int64(3), mustInt(t, replicas))

				t.Log("✓ Q3: Module with extra 'values' field loads successfully")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := cuecontext.New()

			// Load the module directory
			absPath, err := filepath.Abs(tt.modulePath)
			require.NoError(t, err, "failed to get absolute path")

			buildInstances := load.Instances([]string{"."}, &load.Config{
				Dir: absPath,
			})
			require.Len(t, buildInstances, 1, "expected one build instance")

			inst := buildInstances[0]
			require.NoError(t, inst.Err, "instance should not have errors")

			// Build the value
			v := ctx.BuildInstance(inst)

			if tt.expectError {
				assert.Error(t, v.Err(), "expected an error")
				return
			}

			if tt.checks != nil {
				tt.checks(t, v)
			}
		})
	}
}

// TestModuleAssignedToModuleField tests whether the loaded module can be
// assigned to a field typed as #Module (like #ModuleRelease.#module).
func TestModuleAssignedToModuleField(t *testing.T) {
	tests := []struct {
		name        string
		modulePath  string
		expectError bool
		errorMsg    string
	}{
		{
			name:       "simple_module_as_module_field",
			modulePath: "testdata/simple_module",
		},
		{
			name:       "module_with_values_as_module_field",
			modulePath: "testdata/module_with_values",
			// This is the key test: does the extra "values" field
			// cause a conflict when unified with #Module?
			expectError: true,
			errorMsg:    "field not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := cuecontext.New()

			// Load the catalog to get #Module definition
			catalogPath, err := filepath.Abs("../../../catalog/v1alpha1/core")
			require.NoError(t, err)

			catalogInsts := load.Instances([]string{"."}, &load.Config{
				Dir: catalogPath,
			})
			require.Len(t, catalogInsts, 1)
			catalogInst := catalogInsts[0]
			require.NoError(t, catalogInst.Err)

			catalogVal := ctx.BuildInstance(catalogInst)
			require.NoError(t, catalogVal.Err())

			// Get #Module definition
			moduleDefPath := cue.ParsePath("#Module")
			moduleDef := catalogVal.LookupPath(moduleDefPath)
			require.True(t, moduleDef.Exists(), "#Module definition should exist")

			// Load the test module
			absPath, err := filepath.Abs(tt.modulePath)
			require.NoError(t, err)

			moduleInsts := load.Instances([]string{"."}, &load.Config{
				Dir: absPath,
			})
			require.Len(t, moduleInsts, 1)
			moduleInst := moduleInsts[0]
			require.NoError(t, moduleInst.Err)

			moduleVal := ctx.BuildInstance(moduleInst)
			require.NoError(t, moduleVal.Err())

			t.Logf("Module package name: %s", moduleInst.PkgName)
			t.Logf("Module has fields: %v", listFields(moduleVal))

			// Try to unify the module value with #Module
			unified := moduleDef.Unify(moduleVal)

			if tt.expectError {
				assert.Error(t, unified.Err(), "expected unification to fail")
				if tt.errorMsg != "" {
					assert.Contains(t, unified.Err().Error(), tt.errorMsg)
				}
				t.Logf("Expected error: %v", unified.Err())
			} else {
				if unified.Err() != nil {
					t.Logf("Unification error: %v", unified.Err())
					t.Logf("Unified value: %s", unified)
				}
				assert.NoError(t, unified.Err(), "unification should succeed")
				t.Log("✓ Module value unifies with #Module definition")
			}
		})
	}
}

// TestModuleReleaseIntegration tests the full flow of using an imported
// module in a #ModuleRelease structure.
func TestModuleReleaseIntegration(t *testing.T) {
	t.Log("Q5: Does the full #ModuleRelease flow work?")

	ctx := cuecontext.New()

	// Load catalog
	catalogPath, err := filepath.Abs("../../../catalog/v1alpha1/core")
	require.NoError(t, err)

	catalogInsts := load.Instances([]string{"."}, &load.Config{
		Dir: catalogPath,
	})
	require.Len(t, catalogInsts, 1)
	catalogVal := ctx.BuildInstance(catalogInsts[0])
	require.NoError(t, catalogVal.Err())

	// Get #ModuleRelease definition
	moduleReleaseDef := catalogVal.LookupPath(cue.ParsePath("#ModuleRelease"))
	require.True(t, moduleReleaseDef.Exists(), "#ModuleRelease should exist")

	// Load the simple module
	modulePath, err := filepath.Abs("testdata/simple_module")
	require.NoError(t, err)

	moduleInsts := load.Instances([]string{"."}, &load.Config{
		Dir: modulePath,
	})
	require.Len(t, moduleInsts, 1)
	moduleVal := ctx.BuildInstance(moduleInsts[0])
	require.NoError(t, moduleVal.Err())

	// Create a ModuleRelease structure
	// We'll use FillPath to inject the module and values
	release := moduleReleaseDef

	// Fill in metadata
	release = release.FillPath(cue.ParsePath("metadata.name"), "test-release")
	release = release.FillPath(cue.ParsePath("metadata.namespace"), "default")

	// Fill in #module (the imported module)
	release = release.FillPath(cue.ParsePath("#module"), moduleVal)
	require.NoError(t, release.Err(), "filling #module should not error")

	// Fill in values (user-provided config)
	userValues := ctx.CompileString(`{
		image: {
			repository: "nginx"
			tag:        "1.25"
		}
		replicas: 5
	}`)
	require.NoError(t, userValues.Err())

	release = release.FillPath(cue.ParsePath("values"), userValues)

	// Check if the release is valid
	if release.Err() != nil {
		t.Logf("Release error: %v", release.Err())
		t.Logf("Release value: %s", release)
	}
	require.NoError(t, release.Err(), "release should be valid")

	// Check that components are resolved
	components := release.LookupPath(cue.ParsePath("components"))
	require.True(t, components.Exists(), "components should exist")

	web := components.LookupPath(cue.ParsePath("web"))
	require.True(t, web.Exists(), "web component should exist")

	// Check that values flowed through to the component
	replicas := web.LookupPath(cue.ParsePath("spec.scaling.count"))
	require.True(t, replicas.Exists(), "replicas should exist in component")
	assert.Equal(t, int64(5), mustInt(t, replicas), "replicas should be 5 (from user values)")

	image := web.LookupPath(cue.ParsePath("spec.container.image.tag"))
	require.True(t, image.Exists(), "image tag should exist")
	assert.Equal(t, "1.25", mustString(t, image), "image tag should be 1.25")

	t.Log("✓ Q5: Full #ModuleRelease flow works!")
}

// Helper functions

func mustString(t *testing.T, v cue.Value) string {
	t.Helper()
	s, err := v.String()
	require.NoError(t, err, "failed to get string value")
	return s
}

func mustInt(t *testing.T, v cue.Value) int64 {
	t.Helper()
	i, err := v.Int64()
	require.NoError(t, err, "failed to get int64 value")
	return i
}

func listFields(v cue.Value) []string {
	var fields []string
	iter, _ := v.Fields(cue.All())
	for iter.Next() {
		fields = append(fields, iter.Selector().String())
	}
	return fields
}

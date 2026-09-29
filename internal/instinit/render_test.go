package instinit

import (
	"testing"

	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustVersion(t *testing.T, path, version string) module.Version {
	t.Helper()
	v, err := module.NewVersion(path, version)
	require.NoError(t, err)
	return v
}

func webAppInput(t *testing.T) Input {
	t.Helper()
	return Input{
		Name:              "web",
		Namespace:         "demo",
		PackageModulePath: DefaultPackageModulePath("web"),
		Module:            mustVersion(t, "opmodel.dev/modules/web_app@v1", "v1.0.4"),
		Core:              mustVersion(t, "opmodel.dev/core@v2", "v2.0.0-alpha.10"),
		Values: []byte(`{
	image:    "nginx:1.27"
	replicas: 2
}`),
		Source: FromDebugValues,
	}
}

func TestRender_Golden(t *testing.T) {
	files, err := Render(webAppInput(t))
	require.NoError(t, err)
	require.Len(t, files, 3)

	assert.Equal(t, `module: "instance.local/web@v0"
language: {
	version: "v0.17.0"
}
deps: {
	"opmodel.dev/core@v2": {
		v: "v2.0.0-alpha.10"
	}
	"opmodel.dev/modules/web_app@v1": {
		v: "v1.0.4"
	}
}
`, string(files[ModuleFile]))

	assert.Equal(t, `package instance

import (
	core "opmodel.dev/core@v2"
	opmModule "opmodel.dev/modules/web_app@v1"
)

core.#ModuleInstance

metadata: {
	name:      "web"
	namespace: "demo"
}

#module: opmModule
`, string(files[InstanceFile]))

	assert.Equal(t, `// Starting values from the module's debugValues, the author's test values.
// Review them before deploying.
package instance

values: {
	image:    "nginx:1.27"
	replicas: 2
}
`, string(files[ValuesFile]))
}

func TestRender_ModulePathOverride(t *testing.T) {
	in := webAppInput(t)
	in.PackageModulePath = "example.com/deploy/web@v0"
	files, err := Render(in)
	require.NoError(t, err)
	assert.Contains(t, string(files[ModuleFile]), `module: "example.com/deploy/web@v0"`)
	assert.NotContains(t, string(files[ModuleFile]), "instance.local")
}

func TestRender_Deterministic(t *testing.T) {
	first, err := Render(webAppInput(t))
	require.NoError(t, err)
	second, err := Render(webAppInput(t))
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestRender_EmptyValues(t *testing.T) {
	in := webAppInput(t)
	in.Values = nil
	in.Source = FromEmpty
	files, err := Render(in)
	require.NoError(t, err)
	assert.Equal(t, `// The module offers no starting values. Fill in what its #config requires;
// opm instance vet names every missing field.
package instance

values: {}
`, string(files[ValuesFile]))
}

func TestRender_QuotesIdentity(t *testing.T) {
	in := webAppInput(t)
	in.Name = `we"b`
	files, err := Render(in)
	require.NoError(t, err)
	assert.Contains(t, string(files[InstanceFile]), `name:      "we\"b"`)
}

func TestRender_RequiresResolvedInput(t *testing.T) {
	for name, mutate := range map[string]func(*Input){
		"no name":        func(in *Input) { in.Name = "" },
		"no namespace":   func(in *Input) { in.Namespace = "" },
		"no module path": func(in *Input) { in.PackageModulePath = "" },
		"no module":      func(in *Input) { in.Module = module.Version{} },
		"no core":        func(in *Input) { in.Core = module.Version{} },
	} {
		t.Run(name, func(t *testing.T) {
			in := webAppInput(t)
			mutate(&in)
			_, err := Render(in)
			assert.Error(t, err)
		})
	}
}

func TestRender_RefusesInvalidPackageModulePath(t *testing.T) {
	in := webAppInput(t)
	in.PackageModulePath = "Not A Path"
	_, err := Render(in)
	assert.Error(t, err)
}

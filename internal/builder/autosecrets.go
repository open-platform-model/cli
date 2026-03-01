package builder

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/internal/core/component"
	"github.com/opmodel/cli/internal/output"
)

const autoSecretsComponentName = "opm-secrets"

// readAutoSecrets reads the autoSecrets field from a fully-evaluated
// #ModuleRelease CUE value. Returns the value and true if it exists and is
// non-empty, or a zero value and false if absent, bottom, or empty.
func readAutoSecrets(result cue.Value) (cue.Value, bool) {
	v := result.LookupPath(cue.ParsePath("autoSecrets"))

	if !v.Exists() {
		return cue.Value{}, false
	}
	if v.Err() != nil {
		return cue.Value{}, false
	}

	// Check if empty struct.
	iter, err := v.Fields()
	if err != nil {
		return cue.Value{}, false
	}
	if !iter.Next() {
		return cue.Value{}, false
	}

	return v, true
}

// loadSecretsSchema loads opmodel.dev/resources/config@v1 from the module's
// dependency cache and extracts the #Secrets definition.
func loadSecretsSchema(ctx *cue.Context, modulePath string) (cue.Value, error) {
	instances := load.Instances([]string{"opmodel.dev/resources/config@v1"}, &load.Config{
		Dir: modulePath,
	})
	if len(instances) == 0 {
		return cue.Value{}, fmt.Errorf("loading opmodel.dev/resources/config@v1: no instances returned")
	}
	if instances[0].Err != nil {
		return cue.Value{}, fmt.Errorf("loading opmodel.dev/resources/config@v1: %w", instances[0].Err)
	}

	configVal := ctx.BuildInstance(instances[0])
	if err := configVal.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building opmodel.dev/resources/config@v1: %w", err)
	}

	secrets := configVal.LookupPath(cue.ParsePath("#Secrets"))
	if !secrets.Exists() {
		return cue.Value{}, fmt.Errorf("#Secrets not found in opmodel.dev/resources/config@v1")
	}

	return secrets, nil
}

// buildOpmSecretsComponent builds the opm-secrets component by starting from
// the #Secrets schema and using FillPath to set metadata.name and
// spec.secrets entries from the autoSecrets data.
func buildOpmSecretsComponent(ctx *cue.Context, secretsSchema, autoSecrets cue.Value) (*component.Component, error) {
	// Start from #Secrets schema and fill metadata.name.
	comp := secretsSchema.FillPath(
		cue.ParsePath("metadata.name"),
		ctx.CompileString(`"`+autoSecretsComponentName+`"`),
	)

	// Fill spec.secrets.<secretName>.data for each entry in autoSecrets.
	iter, err := autoSecrets.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating autoSecrets: %w", err)
	}
	for iter.Next() {
		secretName := iter.Selector().Unquoted()
		dataVal := iter.Value()
		path := cue.ParsePath(fmt.Sprintf("spec.secrets.%q.data", secretName))
		comp = comp.FillPath(path, dataVal)
	}

	if err := comp.Err(); err != nil {
		return nil, fmt.Errorf("building opm-secrets component: %w", err)
	}

	// Wrap in a map and use ExtractComponents to get a proper *Component.
	wrapped := ctx.CompileString("{}")
	wrapped = wrapped.FillPath(cue.MakePath(cue.Str(autoSecretsComponentName)), comp)
	if err := wrapped.Err(); err != nil {
		return nil, fmt.Errorf("wrapping opm-secrets component: %w", err)
	}

	components, err := component.ExtractComponents(wrapped)
	if err != nil {
		return nil, fmt.Errorf("extracting opm-secrets component: %w", err)
	}

	result, ok := components[autoSecretsComponentName]
	if !ok {
		return nil, fmt.Errorf("opm-secrets component not found after extraction")
	}

	return result, nil
}

// injectAutoSecrets reads autoSecrets from the evaluated #ModuleRelease,
// builds an opm-secrets component, and injects it into the components map.
// If autoSecrets is absent or empty, the components map is returned unchanged.
// Returns an error if the user-defined components already contain "opm-secrets".
func injectAutoSecrets(ctx *cue.Context, result cue.Value, modulePath string, components map[string]*component.Component) error {
	autoSecrets, ok := readAutoSecrets(result)
	if !ok {
		output.Debug("no autoSecrets found, skipping injection")
		return nil
	}

	// Check for name collision before building.
	if _, exists := components[autoSecretsComponentName]; exists {
		return fmt.Errorf("component %q is reserved for auto-secret injection; rename your component", autoSecretsComponentName)
	}

	output.Debug("autoSecrets found, building opm-secrets component")

	secretsSchema, err := loadSecretsSchema(ctx, modulePath)
	if err != nil {
		return fmt.Errorf("auto-secrets injection: %w", err)
	}

	opmSecrets, err := buildOpmSecretsComponent(ctx, secretsSchema, autoSecrets)
	if err != nil {
		return fmt.Errorf("auto-secrets injection: %w", err)
	}

	components[autoSecretsComponentName] = opmSecrets

	output.Debug("opm-secrets component injected",
		"secrets", len(opmSecrets.Resources),
	)

	return nil
}

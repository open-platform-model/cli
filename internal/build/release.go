package build

import (
	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/output"
)

// isModuleRelease checks if the loaded value is a ModuleRelease (has concrete components).
// A ModuleRelease has a concrete `components` field, while a Module only has `#components`.
func (l *ModuleLoader) isModuleRelease(value cue.Value) bool {
	// Check for concrete components field
	componentsValue := value.LookupPath(cue.ParsePath("components"))
	if !componentsValue.Exists() {
		return false
	}

	// Verify it's concrete (can be iterated without error)
	_, err := componentsValue.Fields()
	return err == nil
}

// hasDefinitionComponents checks if the value has #components (CUE definition).
func (l *ModuleLoader) hasDefinitionComponents(value cue.Value) bool {
	// Check for #components
	hashComponents := value.LookupPath(cue.ParsePath("#components"))
	if hashComponents.Exists() {
		return true
	}
	// Also check module.#components path
	hashComponents = value.LookupPath(cue.ParsePath("module.#components"))
	return hashComponents.Exists()
}

// extractComponentsFromModule extracts components from either a Module or ModuleRelease.
// For Modules, it extracts from #components.
// For ModuleReleases, it extracts from components.
func (l *ModuleLoader) extractComponentsFromModule(value cue.Value, isRelease bool) ([]*LoadedComponent, error) {
	var components []*LoadedComponent
	var componentsValue cue.Value

	if isRelease {
		// For releases, use concrete components field
		componentsValue = value.LookupPath(cue.ParsePath("components"))
		if !componentsValue.Exists() {
			componentsValue = value.LookupPath(cue.ParsePath("module.components"))
		}
		output.Debug("extracting from release components field")
	} else {
		// For modules, use #components definition
		componentsValue = value.LookupPath(cue.ParsePath("#components"))
		if !componentsValue.Exists() {
			componentsValue = value.LookupPath(cue.ParsePath("module.#components"))
		}
		output.Debug("extracting from module #components definition")
	}

	if !componentsValue.Exists() {
		// No components is valid (empty module)
		output.Debug("no components found")
		return components, nil
	}

	// Iterate over components
	iter, err := componentsValue.Fields()
	if err != nil {
		// This might happen if #components has incomplete/abstract values
		output.Debug("cannot iterate components, checking for CUE errors", "error", err)
		return nil, &ModuleValidationError{
			Message: "cannot extract components - module may have incomplete required fields",
			Cause:   err,
		}
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		compValue := iter.Value()

		// Check if component value has errors (incomplete fields)
		if compValue.Err() != nil {
			return nil, &ModuleValidationError{
				Message:       "component has incomplete required fields",
				ComponentName: name,
				Cause:         compValue.Err(),
			}
		}

		comp, err := l.extractComponent(name, compValue)
		if err != nil {
			return nil, &ModuleValidationError{
				Message:       "failed to extract component",
				ComponentName: name,
				Cause:         err,
			}
		}
		components = append(components, comp)
	}

	output.Debug("extracted components",
		"count", len(components),
		"isRelease", isRelease,
	)

	return components, nil
}

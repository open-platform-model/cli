package cue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"gopkg.in/yaml.v3"
)

var (
	// ErrUnsupportedFormat is returned when a values file has an unsupported format.
	ErrUnsupportedFormat = errors.New("unsupported file format")
)

// ValuesLoader loads values from CUE, YAML, or JSON files.
type ValuesLoader struct {
	ctx *cue.Context
}

// NewValuesLoader creates a new ValuesLoader.
func NewValuesLoader(ctx *cue.Context) *ValuesLoader {
	return &ValuesLoader{ctx: ctx}
}

// LoadFile loads a values file and returns a CUE value.
// Supports .cue, .yaml, .yml, and .json files.
func (l *ValuesLoader) LoadFile(ctx context.Context, path string) (cue.Value, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cue.Value{}, fmt.Errorf("reading values file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".cue":
		return l.loadCUE(data, path)
	case ".yaml", ".yml":
		return l.loadYAML(data)
	case ".json":
		return l.loadJSON(data)
	default:
		return cue.Value{}, fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
	}
}

// LoadMultiple loads multiple values files and unifies them.
// Files are unified in order, with later files taking precedence.
func (l *ValuesLoader) LoadMultiple(ctx context.Context, paths []string) (cue.Value, error) {
	if len(paths) == 0 {
		// Return an empty struct value
		return l.ctx.CompileString("{}"), nil
	}

	var result cue.Value
	for i, path := range paths {
		value, err := l.LoadFile(ctx, path)
		if err != nil {
			return cue.Value{}, fmt.Errorf("loading %s: %w", path, err)
		}

		if i == 0 {
			result = value
		} else {
			result = result.Unify(value)
			if result.Err() != nil {
				return cue.Value{}, fmt.Errorf("unifying %s: %w", path, result.Err())
			}
		}
	}

	return result, nil
}

// loadCUE compiles CUE source code.
func (l *ValuesLoader) loadCUE(data []byte, path string) (cue.Value, error) {
	value := l.ctx.CompileBytes(data, cue.Filename(path))
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("compiling CUE: %w", value.Err())
	}
	return value, nil
}

// loadYAML parses YAML and converts to CUE.
func (l *ValuesLoader) loadYAML(data []byte) (cue.Value, error) {
	var parsed any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return cue.Value{}, fmt.Errorf("parsing YAML: %w", err)
	}

	// Convert to JSON-compatible form, then to CUE
	jsonData, err := json.Marshal(normalizeYAML(parsed))
	if err != nil {
		return cue.Value{}, fmt.Errorf("converting YAML to JSON: %w", err)
	}

	value := l.ctx.CompileBytes(jsonData)
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("converting to CUE: %w", value.Err())
	}
	return value, nil
}

// loadJSON parses JSON.
func (l *ValuesLoader) loadJSON(data []byte) (cue.Value, error) {
	// Validate JSON first
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return cue.Value{}, fmt.Errorf("parsing JSON: %w", err)
	}

	value := l.ctx.CompileBytes(data)
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("compiling JSON: %w", value.Err())
	}
	return value, nil
}

// normalizeYAML converts YAML-specific types to JSON-compatible types.
// YAML allows map keys of any type, but JSON requires strings.
func normalizeYAML(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = normalizeYAML(v)
		}
		return result
	case map[any]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[fmt.Sprintf("%v", k)] = normalizeYAML(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = normalizeYAML(v)
		}
		return result
	default:
		return v
	}
}

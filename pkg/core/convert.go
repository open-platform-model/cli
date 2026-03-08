package core

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// MarshalJSON returns the JSON byte representation of the resource's CUE value.
func (r *Resource) MarshalJSON() ([]byte, error) {
	if err := r.Value.Err(); err != nil {
		return nil, fmt.Errorf("resource %s: cue value error: %w", r.String(), err)
	}
	b, err := r.Value.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("resource %s: marshal json: %w", r.String(), err)
	}
	return b, nil
}

// MarshalYAML returns the YAML byte representation of the resource's CUE value.
// Converts via JSON to ensure consistent field ordering.
func (r *Resource) MarshalYAML() ([]byte, error) {
	j, err := r.MarshalJSON()
	if err != nil {
		return nil, err
	}
	y, err := yaml.JSONToYAML(j)
	if err != nil {
		return nil, fmt.Errorf("resource %s: json to yaml: %w", r.String(), err)
	}
	return y, nil
}

// ToUnstructured converts the resource to a *unstructured.Unstructured.
// Uses JSON as the intermediate format.
func (r *Resource) ToUnstructured() (*unstructured.Unstructured, error) {
	j, err := r.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var obj map[string]any
	if err := json.Unmarshal(j, &obj); err != nil {
		return nil, fmt.Errorf("resource %s: unmarshal to map: %w", r.String(), err)
	}
	return &unstructured.Unstructured{Object: obj}, nil
}

// ToMap converts the resource to a map[string]any.
func (r *Resource) ToMap() (map[string]any, error) {
	j, err := r.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, fmt.Errorf("resource %s: unmarshal to map: %w", r.String(), err)
	}
	return m, nil
}

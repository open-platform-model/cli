package builder

import (
	"fmt"

	"cuelang.org/go/cue"
)

// extractCUEStringMap extracts a map[string]string from a named struct field
// of the given CUE value. If the field does not exist or is not a struct,
// an empty non-nil map is returned with no error.
func extractCUEStringMap(v cue.Value, field string) (map[string]string, error) {
	result := make(map[string]string)

	fieldVal := v.LookupPath(cue.ParsePath(field))
	if !fieldVal.Exists() {
		return result, nil
	}

	iter, err := fieldVal.Fields()
	if err != nil {
		return result, nil //nolint:nilerr // field exists but is not a struct (e.g. open type) — treat as empty
	}

	for iter.Next() {
		key := iter.Selector().Unquoted()
		val := iter.Value()
		str, err := val.String()
		if err != nil {
			return nil, fmt.Errorf("field %s.%s: %w", field, key, err)
		}
		result[key] = str
	}

	return result, nil
}

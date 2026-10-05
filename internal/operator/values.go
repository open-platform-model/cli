package operator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/open-platform-model/library/opm/kernel"
)

// ValuesInput is what the operator instance's values are made from.
type ValuesInput struct {
	// Recorded is the spec.values of the operator's instance record; nil
	// when there is no record or it records no values.
	Recorded map[string]any
	// Files are the -f/--values files, layered over Recorded in order.
	Files []string
	// Reset is --reset-values: Recorded is dropped for this run.
	Reset bool
}

// Values is the merged values of one install, ready for the render.
type Values struct {
	// Merged is the merged values map (empty, never nil, when nothing was
	// given).
	Merged map[string]any
	// FromRecord reports whether recorded values took part, so a rejection
	// can name --reset-values.
	FromRecord bool
	// Source is the merged map as one kernel values source, attributed to
	// where it came from.
	Source kernel.Source
}

// MergeValues layers the -f files, in order, over the recorded values
// (unless Reset): maps merge key by key, and a scalar or list in a later
// source replaces the earlier one. The module's debugValues are never used.
// Each file is a CUE file whose payload is either its top-level `values`
// field or, without one, the whole file; it must be concrete.
func MergeValues(in ValuesInput) (*Values, error) {
	merged := map[string]any{}
	var origins []string
	fromRecord := false
	if !in.Reset && len(in.Recorded) > 0 {
		merged = deepMerge(merged, normalize(in.Recorded))
		fromRecord = true
		origins = append(origins, fmt.Sprintf("values recorded on %s/%s", OperatorNamespace, OperatorInstanceName))
	}
	for _, f := range in.Files {
		m, err := loadValuesFile(f)
		if err != nil {
			return nil, err
		}
		merged = deepMerge(merged, m)
		origins = append(origins, f)
	}

	data, err := json.Marshal(map[string]any{"values": merged})
	if err != nil {
		return nil, fmt.Errorf("encoding the operator's values: %w", err)
	}
	origin := "operator values"
	if len(origins) > 0 {
		origin = strings.Join(origins, " + ")
	}
	return &Values{
		Merged:     merged,
		FromRecord: fromRecord,
		Source:     kernel.Source{Origin: origin, Data: data},
	}, nil
}

// loadValuesFile decodes one values file to a JSON-shaped map.
func loadValuesFile(path string) (map[string]any, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("values file %q: %w", path, err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("reading values file %q: %w", path, err)
	}
	v := cuecontext.New().CompileBytes(data, cue.Filename(abs))
	if err := v.Err(); err != nil {
		return nil, fmt.Errorf("values file %q: %w", path, err)
	}
	if inner := v.LookupPath(cue.ParsePath("values")); inner.Exists() {
		v = inner
	}
	raw, err := v.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("values file %q is not concrete: %w", path, err)
	}
	m, err := decodeJSONMap(raw)
	if err != nil {
		return nil, fmt.Errorf("values file %q does not hold an object: %w", path, err)
	}
	return m, nil
}

// normalize round-trips a map through JSON, so recorded values (decoded from
// the cluster) and file values share one representation.
func normalize(m map[string]any) map[string]any {
	raw, err := json.Marshal(m)
	if err != nil {
		return m
	}
	out, err := decodeJSONMap(raw)
	if err != nil {
		return m
	}
	return out
}

func decodeJSONMap(raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// deepMerge returns base with over laid on top: a key holding a map on both
// sides merges recursively, any other value from over replaces base's.
func deepMerge(base, over map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(over))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		if bm, ok := out[k].(map[string]any); ok {
			if om, ok := v.(map[string]any); ok {
				out[k] = deepMerge(bm, om)
				continue
			}
		}
		out[k] = v
	}
	return out
}

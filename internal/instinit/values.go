package instinit

import (
	"bytes"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"

	"github.com/open-platform-model/library/opm/schema"
)

// emptyValues is the expression written when no source is usable.
var emptyValues = []byte("{}")

// PickValues walks the values ladder over an acquired module's package value
// (0016:D2/D6): debugValues when the whole value is concrete once defaults
// apply, else empty. The winner is serialized with
// Syntax(cue.Final(), cue.Concrete(false)), so defaults resolve.
//
// A debugValues with any field left without a value (a bare type, an
// undefaulted disjunction, or `_`) is not concrete and falls to empty; a
// partly concrete one is not rendered in part.
func PickValues(pkg cue.Value) ([]byte, ValuesSource, error) {
	debug := pkg.LookupPath(schema.DebugValues)
	if !debug.Exists() || !concrete(debug) {
		return emptyValues, FromEmpty, nil
	}
	data, err := format.Node(debug.Syntax(cue.Final(), cue.Concrete(false)))
	if err != nil {
		return nil, "", fmt.Errorf("rendering debugValues: %w", err)
	}
	return data, FromDebugValues, nil
}

// concrete reports whether v is fully concrete once defaults apply.
func concrete(v cue.Value) bool {
	return v.Validate(cue.Final(), cue.Concrete(true)) == nil
}

// IsEmpty reports whether values renders as an empty struct, which the
// report flags whatever source it came from (0016:D6:R3).
func IsEmpty(values []byte) bool {
	return bytes.Equal(bytes.Join(bytes.Fields(values), nil), emptyValues)
}

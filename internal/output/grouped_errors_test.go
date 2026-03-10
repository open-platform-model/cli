package output

import (
	"testing"

	pkgerrors "github.com/opmodel/cli/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestFormatGroupedErrors(t *testing.T) {
	formatted := FormatGroupedErrors([]pkgerrors.GroupedError{
		{Message: "field not allowed", Locations: []pkgerrors.ErrorLocation{{Path: "values.test", File: "values.cue", Line: 10, Column: 2}, {Path: "values.test", File: "values2.cue", Line: 12, Column: 2}}},
		{Message: "conflicting values", Locations: []pkgerrors.ErrorLocation{{Path: "values.media.test", File: "values.cue", Line: 20, Column: 3}}},
	})
	assert.Contains(t, formatted, "field not allowed")
	assert.Contains(t, formatted, "conflicting values")
	assert.NotContains(t, formatted, "3 errors")
}

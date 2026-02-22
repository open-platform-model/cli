package valuesflow

// ---------------------------------------------------------------------------
// Approach C: rogue file validation
//
// Approach C is the guard that runs BEFORE Approach A loading. It scans the
// module directory for any values*.cue file other than "values.cue" and errors
// immediately, before any load.Instances call occurs.
//
// Only "values.cue" is allowed inside a module directory. Environment-specific
// overrides (values_forge.cue, values_prod.cue, etc.) must live outside the
// module directory and be referenced via --values.
//
// Fixture: testdata/rogue_module/ — contains values.cue + values_forge.cue
// Reference: experiments/values-load-isolation/approach_c_test.go
// ---------------------------------------------------------------------------

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApproachC_RogueValuesFile_Errors proves that validateFileList detects
// values_forge.cue in rogue_module and returns an error before any load occurs.
func TestApproachC_RogueValuesFile_Errors(t *testing.T) {
	dir := fixturePath(t, "rogue_module")
	files := cueFilesInDir(t, dir)

	err := validateFileList(files)

	require.Error(t, err, "rogue_module with values_forge.cue must produce an error")
}

// TestApproachC_RogueValuesFile_ErrorNamesTheFile proves that the error message
// from validateFileList names the rogue file explicitly, making it actionable.
func TestApproachC_RogueValuesFile_ErrorNamesTheFile(t *testing.T) {
	dir := fixturePath(t, "rogue_module")
	files := cueFilesInDir(t, dir)

	err := validateFileList(files)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "values_forge.cue",
		"error message must name the rogue file so the author knows what to fix")
}

// TestApproachC_CleanModule_NoRogueFiles proves that values_module (which has
// only the legitimate values.cue) passes validateFileList without error.
func TestApproachC_CleanModule_NoRogueFiles(t *testing.T) {
	dir := fixturePath(t, "values_module")
	files := cueFilesInDir(t, dir)

	err := validateFileList(files)

	assert.NoError(t, err, "values_module with only values.cue must not be flagged as rogue")
}

// TestApproachC_InlineModule_NoValuesFilesAtAll proves that a module with no
// values*.cue files at all (inline_module) also passes validateFileList.
func TestApproachC_InlineModule_NoValuesFilesAtAll(t *testing.T) {
	dir := fixturePath(t, "inline_module")
	files := cueFilesInDir(t, dir)

	err := validateFileList(files)

	assert.NoError(t, err, "inline_module with no values*.cue files must pass validation")
}

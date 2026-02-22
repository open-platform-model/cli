package valuesloadisolation

// ---------------------------------------------------------------------------
// Approach C: Option 3 enforcement — validate single values.cue in module dir
//
// Regardless of which loading approach (A or B) is chosen, add a validation
// step that errors early if the module directory contains any values*.cue file
// other than values.cue. This enforces the design rule:
//
//   "Only values.cue is allowed inside the module directory.
//    Environment-specific files must live outside the module."
//
// This validation is a pre-check, not a load strategy. It combines with A or B.
//
// These tests prove:
//   - Detection of rogue values*.cue files is reliable
//   - The error message is actionable
//   - A module with only values.cue passes validation
//   - A module with no values*.cue files at all also passes
// ---------------------------------------------------------------------------

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateSingleValuesFile checks that the module directory contains at most
// one values*.cue file and that it is named exactly "values.cue".
//
// Returns nil if the constraint is satisfied.
// Returns an error listing all rogue files if violated.
func validateSingleValuesFile(dir string) error {
	entries, err := readDirEntries(dir)
	if err != nil {
		return fmt.Errorf("scanning module directory: %w", err)
	}

	var rogues []string
	for _, name := range entries {
		if isValuesFile(name) && name != "values.cue" {
			rogues = append(rogues, name)
		}
	}

	if len(rogues) == 0 {
		return nil
	}

	return fmt.Errorf(
		"module directory contains %d unexpected values file(s): %s\n"+
			"Only values.cue is allowed inside the module directory.\n"+
			"Move environment-specific files outside the module and use --values to reference them.",
		len(rogues),
		strings.Join(rogues, ", "),
	)
}

// readDirEntries returns the base names of all non-directory entries in dir.
func readDirEntries(dir string) ([]string, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.cue"))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, filepath.Base(e))
	}
	return names, nil
}

// TestApproachC_DetectsRogueValuesFiles proves that the validator catches
// values_forge.cue and values_testing.cue in the test module directory.
func TestApproachC_DetectsRogueValuesFiles(t *testing.T) {
	err := validateSingleValuesFile(modulePath(t))
	require.Error(t, err, "module with multiple values*.cue files should fail validation")

	t.Logf("Validation error (expected): %v", err)

	// Error must mention the rogue filenames
	assert.True(t,
		strings.Contains(err.Error(), "values_forge.cue") || strings.Contains(err.Error(), "values_testing.cue"),
		"error should name the rogue files, got: %v", err,
	)
}

// TestApproachC_CountsRogueFilesCorrectly proves the count in the error is accurate.
func TestApproachC_CountsRogueFilesCorrectly(t *testing.T) {
	err := validateSingleValuesFile(modulePath(t))
	require.Error(t, err)

	// testdata/module has values_forge.cue and values_testing.cue → 2 rogues
	assert.Contains(t, err.Error(), "2 unexpected values file(s)",
		"should report exactly 2 rogue files")
}

// TestApproachC_ErrorMessageIsActionable checks that the error tells the user
// what to do, not just what went wrong.
func TestApproachC_ErrorMessageIsActionable(t *testing.T) {
	err := validateSingleValuesFile(modulePath(t))
	require.Error(t, err)

	msg := err.Error()
	assert.True(t,
		strings.Contains(msg, "Move") || strings.Contains(msg, "outside") || strings.Contains(msg, "--values"),
		"error message should guide the user toward a fix, got: %v", msg,
	)
	t.Logf("Error message:\n%s", msg)
}

// TestApproachC_AllowsOnlyValuesFile proves that a module directory containing
// only values.cue passes validation with no error.
func TestApproachC_AllowsOnlyValuesFile(t *testing.T) {
	// Use a temp dir with just values.cue and module.cue
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "module.cue"), "package main\n")
	writeFile(t, filepath.Join(tmp, "values.cue"), "package main\nvalues: {}\n")

	err := validateSingleValuesFile(tmp)
	assert.NoError(t, err, "module with only values.cue should pass validation")
}

// TestApproachC_AllowsNoValuesFile proves that a module directory with NO
// values*.cue files at all also passes — absence of values.cue is allowed
// (the pipeline will require it or the user must pass --values, handled elsewhere).
func TestApproachC_AllowsNoValuesFile(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "module.cue"), "package main\n")
	// No values.cue at all

	err := validateSingleValuesFile(tmp)
	assert.NoError(t, err, "module with no values*.cue files should pass validation")
}

// TestApproachC_ValuesFileNameMustBeExact proves that a file named
// "values_production.cue" is caught even when values.cue is absent —
// i.e. the check is about the naming pattern, not just count.
func TestApproachC_ValuesFileNameMustBeExact(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "module.cue"), "package main\n")
	writeFile(t, filepath.Join(tmp, "values_production.cue"), "package main\nvalues: {}\n")
	// Note: no values.cue — only a non-default values file

	err := validateSingleValuesFile(tmp)
	require.Error(t, err, "values_production.cue without values.cue should still fail — rogue name")
	assert.Contains(t, err.Error(), "values_production.cue")
	t.Logf("Correctly rejected: %v", err)
}

// TestApproachC_CombinedWithApproachA shows that the validation + Approach A
// load pipeline together: validate first (catches rogues), then load cleanly.
// When combined, the rogue module fails at validation before load is attempted.
func TestApproachC_CombinedWithApproachA(t *testing.T) {
	dir := modulePath(t)

	// Step 1: validate — this should fail on our rogue-filled testdata
	err := validateSingleValuesFile(dir)
	require.Error(t, err, "validation must catch rogue files")
	t.Logf("Validation correctly rejected module: %v", err)

	// Step 2: if we bypass validation (for test purposes), Approach A still loads cleanly
	_, baseVal, _ := approachALoad(t, dir)
	assert.NoError(t, baseVal.Err(), "Approach A loads cleanly regardless of rogue files (validation is the gate)")
}

// writeFile is a test helper to create a file with given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0o600)
	require.NoError(t, err)
}

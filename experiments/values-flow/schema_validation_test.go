package valuesflow

// ---------------------------------------------------------------------------
// Schema validation: validateAgainstConfig()
//
// After selectValues() determines which values to use, those values are
// validated against the module's #config schema. This happens AFTER selection,
// which means:
//   - Only the values actually in use are validated
//   - Module defaults that were replaced by user values are never validated
//   - Invalid user values are caught even if the defaults were valid
//
// This mirrors builder Step 4: mod.Config.Unify(selectedValues).
//
// Fixture: testdata/values_module/
//   #config: { image: string, replicas: int & >=1 }
//
// User overrides: testdata/user_values.cue (valid), testdata/invalid_values.cue
//   invalid: replicas: 0  (violates int & >=1)
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaValidation_ValidValuesPasses proves that values satisfying all
// #config constraints produce no error from validateAgainstConfig.
func TestSchemaValidation_ValidValuesPasses(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "values_module"))

	selected, _ := selectValues(t, ctx, moduleVal, defaultVals, fixturePath(t, "user_values.cue"))
	config := moduleVal.LookupPath(cue.ParsePath("#config"))

	err := validateAgainstConfig(config, selected)

	assert.NoError(t, err, "valid user values must satisfy #config constraints")
}

// TestSchemaValidation_InvalidValuesFails proves that values violating a #config
// constraint (replicas: 0 < 1) return an error from validateAgainstConfig.
func TestSchemaValidation_InvalidValuesFails(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "values_module"))

	selected, _ := selectValues(t, ctx, moduleVal, defaultVals, fixturePath(t, "invalid_values.cue"))
	config := moduleVal.LookupPath(cue.ParsePath("#config"))

	err := validateAgainstConfig(config, selected)

	require.Error(t, err,
		"replicas:0 must violate the 'int & >=1' constraint declared in #config")
}

// TestSchemaValidation_OnlySelectedValuesValidated proves that validation fires
// on the selected values only. When the user provides invalid values, those are
// what fail — the (valid) module defaults are irrelevant and never checked.
func TestSchemaValidation_OnlySelectedValuesValidated(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "values_module"))
	config := moduleVal.LookupPath(cue.ParsePath("#config"))

	// Defaults are valid on their own — validation must pass.
	defaultSelected, _ := selectValues(t, ctx, moduleVal, defaultVals, "")
	assert.NoError(t, validateAgainstConfig(config, defaultSelected),
		"module defaults must be valid against #config")

	// User provides invalid values — only THOSE are validated; defaults are not.
	invalidSelected, _ := selectValues(t, ctx, moduleVal, defaultVals, fixturePath(t, "invalid_values.cue"))
	assert.Error(t, validateAgainstConfig(config, invalidSelected),
		"invalid user values must fail validation; the (valid) defaults are irrelevant once user values are selected")
}

// TestSchemaValidation_DefaultsAreValidAgainstConfig proves that module author
// defaults (values.cue) satisfy the module's own #config schema. This is a
// sanity check on the fixture — ensures test data is internally consistent.
func TestSchemaValidation_DefaultsAreValidAgainstConfig(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "values_module"))

	defaults, ok := selectValues(t, ctx, moduleVal, defaultVals, "")
	require.True(t, ok)

	config := moduleVal.LookupPath(cue.ParsePath("#config"))

	assert.NoError(t, validateAgainstConfig(config, defaults),
		"module author defaults in values.cue must satisfy #config — fixture sanity check")
}

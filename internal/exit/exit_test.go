package exit

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExitCodeConstants(t *testing.T) {
	assert.Equal(t, 0, ExitSuccess)
	assert.Equal(t, 1, ExitGeneralError)
	assert.Equal(t, 2, ExitValidationError)
	assert.Equal(t, 3, ExitConnectivityError)
	assert.Equal(t, 4, ExitPermissionDenied)
	assert.Equal(t, 5, ExitNotFound)
}

func TestExitError(t *testing.T) {
	inner := fmt.Errorf("something failed")
	e := &ExitError{Code: ExitValidationError, Err: inner}

	assert.Equal(t, "something failed", e.Error())
	assert.Equal(t, inner, e.Unwrap())
}

func TestExitErrorNoInner(t *testing.T) {
	e := &ExitError{Code: ExitGeneralError}
	assert.Contains(t, e.Error(), "exit code 1")
}

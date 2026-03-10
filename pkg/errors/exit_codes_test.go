package errors

import (
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

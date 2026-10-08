package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// A usage error exits 1, the code the specs and 'opm --help' name for it.
// Exit 2 is taken: it means a validation error or a refusal, and for
// 'opm instance status' a resource that is not ready.
func TestRun_UsageErrorExitsOne(t *testing.T) {
	// Keep the user's real config out of the commands that load one.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPM_CONFIG", "")

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"unknown command", []string{"nosuchcommand"}, `unknown command "nosuchcommand" for "opm"`},
		{"unknown flag", []string{"version", "--bogus"}, "unknown flag: --bogus"},
		{"missing argument", []string{"instance", "build"}, "accepts 1 arg(s), received 0"},
		{"too many arguments", []string{"module", "template", "list", "extra"}, `unknown command "extra" for "opm module template list"`},
		{"flag value of the wrong type", []string{"instance", "delete", "x", "--timeout", "soon"}, `invalid argument "soon" for "--timeout" flag`},
		{"flags that exclude each other", []string{"module", "vet", "--name", "a", "--instance-name", "b"}, "[instance-name name] were all set"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := Run(tt.args, &stderr)

			assert.Equal(t, 1, code)
			assert.Contains(t, stderr.String(), tt.wantErr)
		})
	}
}

func TestRun_SuccessExitsZero(t *testing.T) {
	var stderr bytes.Buffer
	assert.Equal(t, 0, Run([]string{"version"}, &stderr))
	assert.Empty(t, stderr.String())
}

// The code of an ExitError passes through, and a printed one is not
// printed twice.
func TestExitCodeOf(t *testing.T) {
	var stderr bytes.Buffer
	assert.Equal(t, 5, exitCode(&opmexit.ExitError{Code: 5, Err: errors.New("no such instance")}, &stderr))
	assert.Equal(t, "no such instance\n", stderr.String())

	stderr.Reset()
	assert.Equal(t, 2, exitCode(&opmexit.ExitError{Code: 2, Err: errors.New("refused"), Printed: true}, &stderr))
	assert.Empty(t, stderr.String())

	stderr.Reset()
	assert.Equal(t, 1, exitCode(errors.New("plain"), &stderr))
	assert.Equal(t, "plain\n", stderr.String())

	assert.Equal(t, 0, exitCode(nil, &stderr))
}

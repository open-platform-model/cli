package exit

import "fmt"

// Exit codes per contracts/exit-codes.md.
const (
	ExitSuccess           = 0
	ExitGeneralError      = 1
	ExitValidationError   = 2
	ExitConnectivityError = 3
	ExitPermissionDenied  = 4
	ExitNotFound          = 5
)

// ExitError wraps an error with an exit code.
// Used by commands to signal specific exit codes to the CLI runner.
type ExitError struct {
	Code    int
	Err     error
	Printed bool // error was already printed by the command layer
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

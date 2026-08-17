package cmdutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/publish"
)

// TestPublishError_ExitCodes pins the pipeline-error → exit-code mapping:
// registry unreachability is 3, anything else unexpected is 1.
func TestPublishError_ExitCodes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "connectivity maps to ExitConnectivityError",
			err:  &publish.ConnectivityError{Op: "listing published versions", Err: errors.New("dial tcp: refused")},
			want: opmexit.ExitConnectivityError,
		},
		{
			name: "wrapped connectivity still maps to ExitConnectivityError",
			err:  fmt.Errorf("publishing: %w", &publish.ConnectivityError{Op: "push", Err: errors.New("timeout")}),
			want: opmexit.ExitConnectivityError,
		},
		{
			// The compatibility walk's mid-flight abort is the same error
			// class as the lookup's: the artifact was never judged, exit 3.
			name: "compat-walk abort maps to ExitConnectivityError",
			err:  &publish.ConnectivityError{Op: "loading example.com/catalogs/demo/resources/v1beta1@v1.0.0", Err: errors.New("dial tcp: refused")},
			want: opmexit.ExitConnectivityError,
		},
		{
			name: "anything else maps to ExitGeneralError",
			err:  errors.New("zipping failed"),
			want: opmexit.ExitGeneralError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := publishError(tt.err)
			var exitErr *opmexit.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, tt.want, exitErr.Code)
		})
	}
}

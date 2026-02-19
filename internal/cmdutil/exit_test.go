package cmdutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	oerrors "github.com/opmodel/cli/internal/errors"
)

func TestExitCodeFromK8sError(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{
			name:     "not found",
			err:      apierrors.NewNotFound(gvr.GroupResource(), "test"),
			wantCode: oerrors.ExitNotFound,
		},
		{
			name:     "forbidden",
			err:      apierrors.NewForbidden(gvr.GroupResource(), "test", nil),
			wantCode: oerrors.ExitPermissionDenied,
		},
		{
			name:     "unauthorized",
			err:      apierrors.NewUnauthorized("test"),
			wantCode: oerrors.ExitPermissionDenied,
		},
		{
			name:     "service unavailable",
			err:      apierrors.NewServiceUnavailable("test"),
			wantCode: oerrors.ExitConnectivityError,
		},
		{
			name:     "other k8s error",
			err:      apierrors.NewBadRequest("test"),
			wantCode: oerrors.ExitGeneralError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExitCodeFromK8sError(tt.err)
			assert.Equal(t, tt.wantCode, got)
		})
	}
}

func TestExitCodeConstants(t *testing.T) {
	// Verify constants match contracts/exit-codes.md
	assert.Equal(t, 0, oerrors.ExitSuccess)
	assert.Equal(t, 1, oerrors.ExitGeneralError)
	assert.Equal(t, 2, oerrors.ExitValidationError)
	assert.Equal(t, 3, oerrors.ExitConnectivityError)
	assert.Equal(t, 4, oerrors.ExitPermissionDenied)
	assert.Equal(t, 5, oerrors.ExitNotFound)
}

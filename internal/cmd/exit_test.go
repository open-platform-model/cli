package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
			wantCode: ExitNotFound,
		},
		{
			name:     "forbidden",
			err:      apierrors.NewForbidden(gvr.GroupResource(), "test", nil),
			wantCode: ExitPermissionDenied,
		},
		{
			name:     "unauthorized",
			err:      apierrors.NewUnauthorized("test"),
			wantCode: ExitPermissionDenied,
		},
		{
			name:     "service unavailable",
			err:      apierrors.NewServiceUnavailable("test"),
			wantCode: ExitConnectivityError,
		},
		{
			name:     "other k8s error",
			err:      apierrors.NewBadRequest("test"),
			wantCode: ExitGeneralError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exitCodeFromK8sError(tt.err)
			assert.Equal(t, tt.wantCode, got)
		})
	}
}

func TestExitCodeConstants(t *testing.T) {
	// Verify constants match contracts/exit-codes.md
	assert.Equal(t, 0, ExitSuccess)
	assert.Equal(t, 1, ExitGeneralError)
	assert.Equal(t, 2, ExitValidationError)
	assert.Equal(t, 3, ExitConnectivityError)
	assert.Equal(t, 4, ExitPermissionDenied)
	assert.Equal(t, 5, ExitNotFound)
}

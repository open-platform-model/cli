package cmdutil_test

import (
	"testing"

	opmexit "github.com/opmodel/cli/internal/exit"

	"github.com/opmodel/cli/internal/cmdutil"
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
			wantCode: opmexit.ExitNotFound,
		},
		{
			name:     "forbidden",
			err:      apierrors.NewForbidden(gvr.GroupResource(), "test", nil),
			wantCode: opmexit.ExitPermissionDenied,
		},
		{
			name:     "unauthorized",
			err:      apierrors.NewUnauthorized("test"),
			wantCode: opmexit.ExitPermissionDenied,
		},
		{
			name:     "service unavailable",
			err:      apierrors.NewServiceUnavailable("test"),
			wantCode: opmexit.ExitConnectivityError,
		},
		{
			name:     "other k8s error",
			err:      apierrors.NewBadRequest("test"),
			wantCode: opmexit.ExitGeneralError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cmdutil.ExitCodeFromK8sError(tt.err)
			assert.Equal(t, tt.wantCode, got)
		})
	}
}

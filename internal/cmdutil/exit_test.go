package cmdutil_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"syscall"
	"testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestExitCodeFromK8sError(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

	// refusedDial is the error client-go returns when the API server port
	// refuses the connection: a *url.Error around a dial *net.OpError.
	refusedDial := &url.Error{
		Op:  "Get",
		URL: "https://127.0.0.1:1/apis/opmodel.dev/v1alpha1/namespaces/default/moduleinstances/x",
		Err: &net.OpError{
			Op:   "dial",
			Net:  "tcp",
			Addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1},
			Err:  os.NewSyscallError("connect", syscall.ECONNREFUSED),
		},
	}

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
			name:     "server timeout",
			err:      apierrors.NewServerTimeout(gvr.GroupResource(), "get", 1),
			wantCode: opmexit.ExitConnectivityError,
		},
		{
			name:     "forbidden wrapped with %w",
			err:      fmt.Errorf("reading instance record: %w", apierrors.NewForbidden(gvr.GroupResource(), "test", nil)),
			wantCode: opmexit.ExitPermissionDenied,
		},
		{
			name:     "refused dial",
			err:      refusedDial,
			wantCode: opmexit.ExitConnectivityError,
		},
		{
			name:     "refused dial wrapped with %w",
			err:      fmt.Errorf("reading instance record: %w", refusedDial),
			wantCode: opmexit.ExitConnectivityError,
		},
		{
			name:     "deadline exceeded wrapped with %w",
			err:      fmt.Errorf("reading instance record: %w", context.DeadlineExceeded),
			wantCode: opmexit.ExitConnectivityError,
		},
		{
			name: "url error around canceled context",
			err: &url.Error{
				Op:  "Get",
				URL: "https://127.0.0.1:6443/api/v1/namespaces/default/pods/x",
				Err: context.Canceled,
			},
			wantCode: opmexit.ExitGeneralError,
		},
		{
			name: "canceled dial holds a net error",
			err: &url.Error{
				Op:  "Get",
				URL: "https://127.0.0.1:6443/api/v1/namespaces/default/pods/x",
				Err: &net.OpError{Op: "dial", Net: "tcp", Err: context.Canceled},
			},
			wantCode: opmexit.ExitGeneralError,
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

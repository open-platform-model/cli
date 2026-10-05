package query

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/cmdutil"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
)

// A Deployment whose Available condition is True but whose rollout is behind
// (updatedReplicas below spec.replicas) is NotReady, and status exits 2.
func TestPrintInstanceStatus_RolloutBehindExitsNotReady(t *testing.T) {
	inv := &inventory.Record{Name: "demo", Namespace: "apps"}
	opts := BuildStatusOptions("apps", &cmdutil.InstanceSelectorFlags{InstanceName: "demo"}, output.FormatJSON, false, inv,
		[]*unstructured.Unstructured{rolledOutDeployment("web", false)}, nil, nil)
	captureLog(t)

	var err error
	out := captureStdout(t, func() {
		err = PrintInstanceStatus(context.Background(), nil, opts, "demo")
	})

	var got struct {
		Resources []struct {
			Kind   string `json:"kind"`
			Status string `json:"status"`
		} `json:"resources"`
		AggregateStatus string `json:"aggregateStatus"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &got), out)
	require.Len(t, got.Resources, 1)
	assert.Equal(t, "Deployment", got.Resources[0].Kind)
	assert.Equal(t, "NotReady", got.Resources[0].Status)
	assert.Equal(t, "NotReady", got.AggregateStatus)

	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	assert.Equal(t, 2, exitErr.Code)
}

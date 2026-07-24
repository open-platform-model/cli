package apply

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/version"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
	pkgmodule "github.com/open-platform-model/cli/pkg/module"
)

// withReleasedCLIVersion overrides the ldflags-set CLI version for the duration
// of a test so the operator-version ceiling gate treats it as a released semver
// (the default "dev" makes the gate skip its Platform probe). The value is newer
// than any Platform the gate tests seed, so the ceiling passes.
func withReleasedCLIVersion(t *testing.T) {
	t.Helper()
	orig := version.Version
	version.Version = "v2.0.0"
	t.Cleanup(func() { version.Version = orig })
}

// The cluster gates must probe in a fixed order — CRD presence, then the CRD
// field floor, then the operator-version ceiling — and short-circuit on the
// first failure so a later probe never runs against an unmet precondition.
func TestRunClusterGates_ProbeOrder(t *testing.T) {
	ctx := context.Background()

	t.Run("all pass: CRD (presence, field floor) then Platform (ceiling)", func(t *testing.T) {
		withReleasedCLIVersion(t) // newer than the operator below, so the ceiling passes
		client, rec := recordingDynamicClient(makeModuleInstanceCRD(true, true), makePlatform("1.0.0"))

		require.NoError(t, RunClusterGates(ctx, client))
		assert.Equal(t,
			[]string{"customresourcedefinitions", "customresourcedefinitions", "platforms"},
			rec.gets,
			"presence and field-floor both read the CRD before the ceiling reads the Platform",
		)
	})

	t.Run("missing CRD short-circuits before field floor and ceiling", func(t *testing.T) {
		withReleasedCLIVersion(t)
		client, rec := recordingDynamicClient() // no CRD, no Platform

		err := RunClusterGates(ctx, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ModuleInstance CRD not found")
		assert.Equal(t, []string{"customresourcedefinitions"}, rec.gets,
			"presence fails first, so the field-floor and ceiling probes never run")
	})

	t.Run("field-floor failure short-circuits before the ceiling", func(t *testing.T) {
		withReleasedCLIVersion(t)
		client, rec := recordingDynamicClient(makeModuleInstanceCRD(false, true)) // CRD present, missing spec.owner

		err := RunClusterGates(ctx, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required fields")
		assert.Equal(t, []string{"customresourcedefinitions", "customresourcedefinitions"}, rec.gets,
			"presence then field-floor read the CRD; the Platform ceiling probe never runs")
	})
}

// Dry-run writes nothing, so the write-protecting cluster gates are skipped
// entirely — a dry-run must succeed against a cluster with no ModuleInstance CRD
// installed and must issue zero gate probes.
func TestExecute_DryRunSkipsClusterGates(t *testing.T) {
	ctx := context.Background()
	withReleasedCLIVersion(t)

	// An empty render isolates the gate decision from the apply-patch path: the
	// flow reaches the gate section and then returns cleanly with nothing to do,
	// so a recorded probe can only have come from the gates themselves.
	newReq := func(client *kubernetes.Client, dryRun bool) Request {
		return Request{
			Result: &workflowrender.Result{
				Resources: nil,
				Instance:  pkgmodule.InstanceMetadata{Name: "demo", Namespace: "default", UUID: ""},
			},
			K8sClient: client,
			Log:       output.InstanceLogger("gate-test"),
			Options:   Options{DryRun: dryRun},
		}
	}

	t.Run("dry-run succeeds and probes nothing on a CRD-less cluster", func(t *testing.T) {
		client, rec := recordingDynamicClient()
		require.NoError(t, Execute(ctx, newReq(client, true)))
		assert.Empty(t, rec.gets, "dry-run must not probe the CRD or Platform")
	})

	t.Run("the same non-dry-run apply DOES run the gates (and fails on the missing CRD)", func(t *testing.T) {
		client, rec := recordingDynamicClient()
		err := Execute(ctx, newReq(client, false))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ModuleInstance CRD not found")
		assert.Equal(t, []string{"customresourcedefinitions"}, rec.gets,
			"a real apply probes the CRD — proving the dry-run zero-probe result is the exemption, not an empty flow")
	})
}

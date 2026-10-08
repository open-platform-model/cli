package instance

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

// shortPoll makes the wait loops poll fast for the test.
func shortPoll(t *testing.T) {
	t.Helper()
	prev := kubernetes.WaitPollInterval
	kubernetes.WaitPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { kubernetes.WaitPollInterval = prev })
}

// holdConfigMapsWithFinalizer keeps every ConfigMap after its accepted
// delete and shows it with a finalizer.
func (s *claimScenario) holdConfigMapsWithFinalizer(t *testing.T, finalizer string) {
	t.Helper()
	cm, err := s.fake.Tracker().Get(configMapGVR, "apps", "web")
	require.NoError(t, err)
	accessor, ok := cm.(interface{ SetFinalizers([]string) })
	require.True(t, ok)
	accessor.SetFinalizers([]string{finalizer})
	require.NoError(t, s.fake.Tracker().Update(configMapGVR, cm, "apps"))
	s.holdConfigMaps()
}

// --wait and --timeout reach the delete; --wait is off by default and
// --timeout defaults to five minutes.
func TestInstanceDeleteCmd_WaitFlag(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantWait    bool
		wantTimeout time.Duration
	}{
		{"default does not wait", []string{"jellyfin"}, false, 5 * time.Minute},
		{"--yes does not wait", []string{"jellyfin", "--yes"}, false, 5 * time.Minute},
		{"--wait", []string{"jellyfin", "--wait"}, true, 5 * time.Minute},
		{"--wait --timeout", []string{"jellyfin", "--wait", "--timeout", "30s"}, true, 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got deleteFlags
			orig := runDelete
			t.Cleanup(func() { runDelete = orig })
			runDelete = func(_ context.Context, _ string, _ *config.GlobalConfig, _ *cmdutil.K8sFlags, _ string, flags deleteFlags) error {
				got = flags
				return nil
			}
			cmd := NewInstanceDeleteCmd(&config.GlobalConfig{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tt.args)
			require.NoError(t, cmd.Execute())
			assert.Equal(t, tt.wantWait, got.Wait)
			assert.Equal(t, tt.wantTimeout, got.Timeout)
		})
	}

	cmd := NewInstanceDeleteCmd(&config.GlobalConfig{})
	wait := cmd.Flags().Lookup("wait")
	require.NotNil(t, wait)
	assert.Equal(t, "false", wait.DefValue)
	assert.Empty(t, wait.Shorthand)
	assert.Contains(t, wait.Usage, "--dry-run")
	assert.Contains(t, wait.Usage, "operator-managed")
	assert.Contains(t, cmd.Flags().Lookup("timeout").Usage, "--wait")
	assert.Contains(t, cmd.Long, "exits 1", "the help states the exit code of a timeout")
	assert.Contains(t, cmd.Long, "5m0s", "and the default timeout")
}

// A resource that a finalizer holds makes --wait time out: the command names
// the resource and its finalizer, prints no success line, keeps the
// ModuleInstance and exits 1. The kept claim is not waited for and is not
// listed as terminating.
func TestConfirmAndDelete_WaitTimesOutOnAHeldResource(t *testing.T) {
	shortPoll(t)
	s := newClaimScenario()
	s.holdConfigMapsWithFinalizer(t, "example.io/hold")

	out, err := s.confirm(t, "demo", deleteFlags{SkipConfirm: true, Wait: true, Timeout: 40 * time.Millisecond}, "")

	requireExitCode(t, err, opmexit.ExitGeneralError)
	terminating := lineWith(out, "ConfigMap/apps/web", output.StatusTerminating)
	require.NotEmpty(t, terminating, "the held resource is listed as terminating:\n%s", out)
	assert.Contains(t, terminating, "WARN")
	assert.Contains(t, terminating, "example.io/hold", "with the finalizer that holds it")
	assert.Empty(t, lineWith(out, "PersistentVolumeClaim/apps/data", output.StatusTerminating), "a kept claim is not waited for")
	assert.Contains(t, out, "waiting for 1 deleted resource(s) to be gone")
	assert.Contains(t, out, "1 deleted resource(s) are still terminating")
	assert.Contains(t, out, "The ModuleInstance was kept")
	assert.Contains(t, out, "Without --wait")
	assert.NotContains(t, out, "Instance deleted")
	assert.NotContains(t, out, "all resources have been deleted")
	assert.True(t, s.exists(inventory.ModuleInstanceGVR, "demo"), "the ModuleInstance still tracks the resource")
}

// After a timeout the same command runs again: it finds the instance, and
// once the cause is gone it waits to the end, deletes the ModuleInstance and
// passes. A run without --wait would have deleted the ModuleInstance at once.
func TestConfirmAndDelete_WaitReRunAfterATimeout(t *testing.T) {
	shortPoll(t)
	s := newClaimScenario()
	s.holdConfigMapsWithFinalizer(t, "example.io/hold")
	flags := deleteFlags{SkipConfirm: true, Wait: true, Timeout: 40 * time.Millisecond}

	_, err := s.confirm(t, "demo", flags, "")
	requireExitCode(t, err, opmexit.ExitGeneralError)

	// The finalizer's controller lets go: deletes go through again.
	s.fake.PrependReactor("delete", "configmaps", func(a k8stesting.Action) (bool, runtime.Object, error) {
		d, ok := a.(k8stesting.DeleteAction)
		require.True(t, ok)
		return true, nil, s.fake.Tracker().Delete(configMapGVR, d.GetNamespace(), d.GetName())
	})

	out, err := s.confirm(t, "demo", flags, "")
	require.NoError(t, err, out)
	assert.Contains(t, out, "Instance deleted")
	assert.False(t, s.exists(configMapGVR, "web"))
	assert.False(t, s.exists(inventory.ModuleInstanceGVR, "demo"))
}

// When the deleted resources go, --wait reports as a run without it does,
// with one more line that says it waited. The kept claim still exists and
// does not hold the command.
func TestConfirmAndDelete_WaitCompletes(t *testing.T) {
	shortPoll(t)
	s := newClaimScenario()

	out, err := s.confirm(t, "demo", deleteFlags{SkipConfirm: true, Wait: true, Timeout: time.Minute}, "")

	require.NoError(t, err, out)
	assert.Equal(t, `INFO m:demo: deleting resources in namespace "apps"
INFO m:demo: r:ConfigMap/apps/web                              - deleted
INFO m:demo: r:PersistentVolumeClaim/apps/data                 = kept
INFO m:demo: waiting for 1 deleted resource(s) to be gone timeout=1m0s
`+noWaitOutput[len(`INFO m:demo: deleting resources in namespace "apps"
INFO m:demo: r:ConfigMap/apps/web                              - deleted
INFO m:demo: r:PersistentVolumeClaim/apps/data                 = kept
`):], logClock.ReplaceAllString(out, ""))
	assert.True(t, s.exists(claimGVR, "data"), "the kept claim is still there")
	assert.False(t, s.exists(inventory.ModuleInstanceGVR, "demo"))
}

// A dry run sends no delete, so --wait has nothing to wait for and the
// output is that of a dry run without it.
func TestConfirmAndDelete_WaitDoesNothingOnADryRun(t *testing.T) {
	shortPoll(t)
	plain := newClaimScenario()
	want, err := plain.confirm(t, "demo", deleteFlags{DryRun: true}, "")
	require.NoError(t, err)

	s := newClaimScenario()
	s.holdConfigMapsWithFinalizer(t, "example.io/hold")
	out, err := s.confirm(t, "demo", deleteFlags{DryRun: true, Wait: true, Timeout: 40 * time.Millisecond}, "")

	require.NoError(t, err, out)
	assert.Equal(t, logClock.ReplaceAllString(want, ""), logClock.ReplaceAllString(out, ""))
	assert.True(t, s.exists(inventory.ModuleInstanceGVR, "demo"))
}

// For an operator-managed instance --wait changes nothing: the command
// deletes the ModuleInstance and waits for it to be gone, as without it.
func TestConfirmAndDelete_WaitChangesNothingForAnOperatorManagedInstance(t *testing.T) {
	shortPoll(t)
	run := func(wait bool) (string, []string) {
		s := newClaimScenario()
		s.operatorManaged(t, true, dataPolicyDelete)
		s.installOperator(t, moduleInstanceCRD(true))
		s.fake.ClearActions()
		out, err := s.confirm(t, "demo", deleteFlags{SkipConfirm: true, Wait: wait, Timeout: 5 * time.Second}, "")
		require.NoError(t, err, out)
		assert.False(t, s.exists(inventory.ModuleInstanceGVR, "demo"))
		return logClock.ReplaceAllString(out, ""), s.requests()
	}

	wantOut, wantRequests := run(false)
	gotOut, gotRequests := run(true)

	assert.Equal(t, wantOut, gotOut)
	assert.Equal(t, wantRequests, gotRequests)
	assert.Contains(t, gotOut, "waiting for the operator to finish cleanup")
}

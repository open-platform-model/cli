package instance

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/inventory"
)

// logClock matches the time a log line starts with.
var logClock = regexp.MustCompile(`(?m)^\d{2}:\d{2}:\d{2} `)

// holdConfigMaps makes the scenario's cluster accept every ConfigMap delete
// and keep the object, as the API server does while a finalizer holds it.
func (s *claimScenario) holdConfigMaps() {
	s.fake.PrependReactor("delete", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
}

// requests lists what the scenario's cluster received, one "verb resource"
// per request, in order.
func (s *claimScenario) requests() []string {
	actions := s.fake.Actions()
	out := make([]string, 0, len(actions))
	for _, a := range actions {
		out = append(out, a.GetVerb()+" "+a.GetResource().Resource)
	}
	return out
}

// noWaitOutput and noWaitRequests are what `opm instance delete --yes`
// printed and sent, before --wait existed, for an instance whose ConfigMap
// stays after its accepted delete. They pin that a run without --wait is
// unchanged: the same bytes (the clock of each log line apart), the same
// requests (no read after the delete), and exit 0. This file also compiles and
// passes on the commit before --wait, which is how the pin was taken.
const noWaitOutput = `INFO m:demo: deleting resources in namespace "apps"
INFO m:demo: r:ConfigMap/apps/web                              - deleted
INFO m:demo: r:PersistentVolumeClaim/apps/data                 = kept
✔ Instance deleted

Kept 1 PersistentVolumeClaim(s) and the data on them. OPM no longer tracks them.
Applying the instance again takes them back. To delete a claim and its data:
  kubectl delete pvc data -n apps
To delete claims together with an instance, pass --delete-data.
`

var noWaitRequests = []string{
	"get moduleinstances",
	"get persistentvolumeclaims",
	"get configmaps",
	"get configmaps",
	"delete configmaps",
	"delete moduleinstances",
}

func TestConfirmAndDelete_WithoutWaitIsUnchanged(t *testing.T) {
	s := newClaimScenario()
	s.holdConfigMaps()

	out, err := s.confirm(t, "demo", deleteFlags{SkipConfirm: true}, "")

	require.NoError(t, err, "a resource that still exists after its accepted delete is no failure")
	assert.Equal(t, noWaitOutput, logClock.ReplaceAllString(out, ""))
	assert.Equal(t, noWaitRequests, s.requests())
	assert.True(t, s.exists(configMapGVR, "web"), "the ConfigMap is still there")
	assert.False(t, s.exists(inventory.ModuleInstanceGVR, "demo"), "and the ModuleInstance is deleted all the same")
}

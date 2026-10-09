package inventory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/k8s/ownership"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
)

const (
	guardSelf  = "11111111-1111-1111-1111-111111111111"
	guardOther = "22222222-2222-2222-2222-222222222222"
)

// guardCM is a live ConfigMap in "default" with the given labels and adopt
// annotation ("" for none).
func guardCM(name string, labels map[string]string, adopt string) *unstructured.Unstructured {
	obj := liveObject("v1", "ConfigMap", "default", name)
	obj.SetLabels(labels)
	if adopt != "" {
		obj.SetAnnotations(map[string]string{opmlabels.AnnotationAdopt: adopt})
	}
	return obj
}

func guardEntry(name string) k8sinventory.Entry {
	return k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: name}
}

func managedBy(actor, uuid string) map[string]string {
	l := map[string]string{opmlabels.ManagedBy: actor}
	if uuid != "" {
		l[opmlabels.ModuleInstanceUUID] = uuid
	}
	return l
}

func terminating(obj *unstructured.Unstructured) *unstructured.Unstructured {
	now := metav1.Now()
	obj.SetDeletionTimestamp(&now)
	obj.SetFinalizers([]string{"foregroundDeletion"})
	return obj
}

// One row per answer of the library's apply verdict, for an object outside
// the inventory and for one inside it. The guard adds no rule of its own.
func TestGuard_Verdicts(t *testing.T) {
	tests := []struct {
		name        string
		live        *unstructured.Unstructured
		recorded    bool
		refuseLetGo bool
		wantRefuse  ownership.ApplyRefusal
		wantLetGo   bool
		wantManaged bool
		wantText    []string
	}{
		{name: "absent object passes"},
		{
			name: "foreign object is refused and the message names the adopt annotation", live: guardCM("cm", nil, ""),
			wantRefuse: ownership.RefuseForeignObject,
			wantText:   []string{"ConfigMap/default/cm", "not managed by OPM", opmlabels.AnnotationAdopt + "=" + guardSelf},
		},
		{
			name: "object of another instance is refused", live: guardCM("cm", managedBy(opmlabels.ManagedByCLI, guardOther), ""),
			wantRefuse: ownership.RefuseOtherInstance, wantText: []string{guardOther},
		},
		{
			name: "this instance's own object passes and is reported as unrecorded", live: guardCM("cm", managedBy(opmlabels.ManagedByCLI, guardSelf), ""),
			wantManaged: true,
		},
		{
			name: "own object labeled by an older cli (legacy managed-by value) passes", live: guardCM("cm", managedBy(opmlabels.ManagedByLegacy, guardSelf), ""),
			wantManaged: true,
		},
		{
			name: "OPM-managed object without a UUID label passes", live: guardCM("cm", managedBy(opmlabels.ManagedByLegacy, ""), ""),
			wantManaged: true,
		},
		{
			name: "recorded object passes whatever its UUID label", live: guardCM("cm", managedBy(opmlabels.ManagedByCLI, guardOther), ""),
			recorded: true,
		},
		{
			name: "recorded object without OPM labels passes", live: guardCM("cm", nil, ""),
			recorded: true,
		},
		{
			name: "terminating object outside the inventory is refused", live: terminating(guardCM("cm", managedBy(opmlabels.ManagedByCLI, guardSelf), "")),
			wantRefuse: ownership.RefuseTerminating, wantText: []string{"is being deleted"},
		},
		{
			name: "terminating recorded object is refused", live: terminating(guardCM("cm", managedBy(opmlabels.ManagedByCLI, guardSelf), "")),
			recorded: true, wantRefuse: ownership.RefuseTerminating,
		},
		{
			name: "adopt annotation naming this instance lifts the foreign refusal", live: guardCM("cm", nil, guardSelf),
		},
		{
			name: "adopt annotation naming this instance lifts the other-instance refusal", live: guardCM("cm", managedBy(opmlabels.ManagedByCLI, guardOther), guardSelf),
			wantManaged: true,
		},
		{
			name: "adopt annotation does not lift the terminating refusal", live: terminating(guardCM("cm", nil, guardSelf)),
			wantRefuse: ownership.RefuseTerminating,
		},
		{
			name: "recorded object adopted by another instance is let go", live: guardCM("cm", managedBy(opmlabels.ManagedByCLI, guardSelf), guardOther),
			recorded: true, wantLetGo: true,
		},
		{
			name: "unrecorded OPM-managed object adopted by another instance is let go", live: guardCM("cm", managedBy(opmlabels.ManagedByCLI, guardOther), guardOther),
			wantLetGo: true,
		},
		{
			name: "with RefuseLetGo an adopted-elsewhere object is a refusal", live: guardCM("cm", managedBy(opmlabels.ManagedByCLI, guardSelf), guardOther),
			recorded: true, refuseLetGo: true, wantRefuse: ownership.RefuseAdoptedElsewhere, wantText: []string{guardOther},
		},
		{
			name: "this instance's UUID without an OPM managed-by label is still foreign", live: guardCM("cm", map[string]string{"app.kubernetes.io/managed-by": "kustomize", opmlabels.ModuleInstanceUUID: guardSelf}, ""),
			wantRefuse: ownership.RefuseForeignObject, wantText: []string{opmlabels.AnnotationAdopt + "=" + guardSelf},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var objs []runtime.Object
			if tc.live != nil {
				objs = append(objs, tc.live)
			}
			dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), objs...)
			client := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dyn}
			in := GuardInput{
				Entries:      []k8sinventory.Entry{guardEntry("cm")},
				InstanceUUID: guardSelf,
				RefuseLetGo:  tc.refuseLetGo,
			}
			if tc.recorded {
				// The record names the object at another version and component.
				in.Previous = []k8sinventory.Entry{{Version: "v1beta1", Kind: "ConfigMap", Namespace: "default", Name: "cm", Component: "old"}}
			}

			result, err := Guard(context.Background(), client, in)

			if tc.wantRefuse == "" {
				require.NoError(t, err)
			} else {
				var refusal *GuardRefusalError
				require.ErrorAs(t, err, &refusal)
				require.Len(t, refusal.Refused, 1)
				assert.Equal(t, tc.wantRefuse, refusal.Refused[0].Reason)
				assert.True(t, refusal.Has(tc.wantRefuse))
				for _, s := range tc.wantText {
					assert.Contains(t, err.Error(), s)
				}
				assert.NotContains(t, err.Error(), "--force", "no flag lifts the guard")
			}
			assert.Equal(t, tc.wantLetGo, len(result.LetGo) == 1, "let go")
			if tc.wantLetGo {
				assert.Contains(t, result.LetGo[0].Message, guardOther)
			}
			assert.Equal(t, tc.wantManaged, len(result.Managed) == 1, "reported as existing under OPM management")
			for _, a := range dyn.Actions() {
				assert.Equal(t, "get", a.GetVerb(), "the guard only reads")
			}
		})
	}
}

// Every refused object is reported, in entry order, and the objects the
// guard allows are still reported to a caller that only looks.
func TestGuard_ReportsEveryRefusal(t *testing.T) {
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(),
		guardCM("foreign-a", nil, ""),
		guardCM("mine", managedBy(opmlabels.ManagedByCLI, guardSelf), ""),
		guardCM("foreign-b", nil, ""),
	)
	client := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dyn}

	result, err := Guard(context.Background(), client, GuardInput{
		Entries:      []k8sinventory.Entry{guardEntry("foreign-a"), guardEntry("mine"), guardEntry("absent"), guardEntry("foreign-b")},
		InstanceUUID: guardSelf,
	})

	var refusal *GuardRefusalError
	require.ErrorAs(t, err, &refusal)
	require.Len(t, refusal.Refused, 2)
	assert.Equal(t, "foreign-a", refusal.Refused[0].Entry.Name)
	assert.Equal(t, "foreign-b", refusal.Refused[1].Entry.Name)
	assert.Contains(t, err.Error(), "2 object(s)")
	assert.Equal(t, []k8sinventory.Entry{guardEntry("mine")}, result.Managed)
}

// The guard refuses an object it cannot read, recorded or not;
// only a NotFound answer proves the name is free.
func TestGuard_UnreadableObjectRefuses(t *testing.T) {
	ctx := context.Background()
	entry := guardEntry("taken")
	forbidden := apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "taken", errors.New("no access"))

	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, forbidden
	})
	client := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dyn}

	_, err := Guard(ctx, client, GuardInput{Entries: []k8sinventory.Entry{entry}, InstanceUUID: guardSelf})
	require.Error(t, err, "an unreadable object is refused")
	assert.Contains(t, err.Error(), "ConfigMap/taken")
	assert.Contains(t, err.Error(), `"default"`)
	assert.True(t, apierrors.IsForbidden(err), "the read error stays in the chain")
	var refusal *GuardRefusalError
	assert.False(t, errors.As(err, &refusal), "a failed read is not an ownership refusal")

	_, err = Guard(ctx, client, GuardInput{
		Entries: []k8sinventory.Entry{entry}, Previous: []k8sinventory.Entry{entry}, InstanceUUID: guardSelf,
	})
	require.Error(t, err, "the record does not pass an unreadable object")

	absent := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())}
	_, err = Guard(ctx, absent, GuardInput{Entries: []k8sinventory.Entry{entry}, InstanceUUID: guardSelf})
	require.NoError(t, err, "an absent object passes")
}

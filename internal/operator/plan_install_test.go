package operator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/modref"
	"github.com/open-platform-model/cli/internal/operator/operatortest"
	"github.com/open-platform-model/cli/internal/publish"
)

func TestResolveTarget(t *testing.T) {
	reg := operatortest.Registry(t,
		operatortest.Version{Module: PinnedModuleVersion, Operator: PinnedOperatorVersion[1:]},
		operatortest.Version{Module: "0.2.0", Operator: "1.0.0-beta.8"},
		operatortest.Version{Module: "0.3.0", OperatorPackage: "-"},
	)
	src := operatortest.Source(t, reg)
	ctx := context.Background()

	// "Default install uses the pin".
	res, target, err := ResolveTarget(ctx, src, reg, "")
	require.NoError(t, err)
	assert.Equal(t, "v"+PinnedModuleVersion, res.Version)
	assert.Equal(t, Target{ModuleVersion: "v" + PinnedModuleVersion, OperatorVersion: PinnedOperatorVersion, Default: true}, target)

	// "Selecting another version", pinned and floating.
	_, target, err = ResolveTarget(ctx, src, reg, "0.2.0")
	require.NoError(t, err)
	assert.Equal(t, Target{ModuleVersion: "v0.2.0", OperatorVersion: "v1.0.0-beta.8"}, target)

	// A major floats to its newest release, 0.3.0, which states no
	// operator version and is refused.
	_, _, err = ResolveTarget(ctx, src, reg, "v0")
	var verr *VersionError
	require.ErrorAs(t, err, &verr)
	assert.Contains(t, err.Error(), "opm_operator 0.3.0")

	// "Old-style operator tag", with and without its "v".
	for _, old := range []string{"v1.0.0-beta.5", "1.0.0-beta.5"} {
		_, _, err = ResolveTarget(ctx, src, reg, old)
		var refusal *modref.RefusalError
		require.ErrorAs(t, err, &refusal, old)
		assert.Contains(t, err.Error(), "--version now takes an operator module version", old)
		assert.Contains(t, err.Error(), old)
	}

	// An unserved module version is a plain refusal.
	_, _, err = ResolveTarget(ctx, src, reg, "0.9.0")
	var refusal *modref.RefusalError
	require.ErrorAs(t, err, &refusal)
	assert.NotContains(t, err.Error(), "release tag")

	// An unreachable registry is a connectivity failure.
	unreachable := "opmodel.dev=127.0.0.1:1+insecure"
	_, _, err = ResolveTarget(ctx, operatortest.Source(t, unreachable), unreachable, "")
	var connErr *publish.ConnectivityError
	require.ErrorAs(t, err, &connErr)
}

func TestLooksLikeOperatorTag(t *testing.T) {
	for _, v := range []string{"v1.0.0-beta.5", "v1.0.0", "1.0.0-beta.5", "v0.1.0"} {
		assert.True(t, looksLikeOperatorTag(v), v)
	}
	for _, v := range []string{"0.1.0", "v0", "latest", "0.2.0-rc.1"} {
		assert.False(t, looksLikeOperatorTag(v), v)
	}
}

// operatorRecord is the operator instance's record as a fixture.
func operatorRecord(owner string, values map[string]any) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "opmodel.dev/v1alpha1",
		"kind":       "ModuleInstance",
		"metadata":   map[string]any{"name": OperatorInstanceName, "namespace": OperatorNamespace},
		"spec": map[string]any{
			"owner":  owner,
			"module": map[string]any{"path": OperatorModulePath + "@v0", "version": "0.1.0"},
		},
		"status": map[string]any{
			"instanceUUID": testInstanceUUID,
			"inventory":    map[string]any{"revision": int64(1), "count": int64(0), "entries": []any{}},
		},
	}}
	if values != nil {
		obj.Object["spec"].(map[string]any)["values"] = values
	}
	return obj
}

// ownerlessRecord is the operator instance's record with no spec.owner,
// which resolves as operator-owned.
func ownerlessRecord() *unstructured.Unstructured {
	obj := operatorRecord("", nil)
	unstructured.RemoveNestedField(obj.Object, "spec", "owner")
	return obj
}

func newEnv(fc *fakeCluster, r *fakeRender) InstallEnv {
	return InstallEnv{Client: fc.client, Render: r.render, CLIVersion: "v1.0.0-beta.9"}
}

var defaultTarget = Target{ModuleVersion: "v0.1.0", OperatorVersion: "v1.0.0-beta.7", Default: true}

func TestPlanInstall_FreshClusterWritesNothing(t *testing.T) {
	fc := newFakeCluster(t)
	r := &fakeRender{objs: moduleObjects(renderOpts{})}

	plan, err := PlanInstall(context.Background(), newEnv(fc, r), testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: time.Second})
	require.NoError(t, err)
	assert.Nil(t, plan.PrevRecord)
	assert.Len(t, plan.CRDs, 4)
	assert.Len(t, plan.Objects(), 8)
	assert.Equal(t, `{"values":{}}`, string(r.values.Data), "debugValues are never used; no values is an empty object")
	assert.Empty(t, fc.Writes())

	plan.CRDsOnly = true
	assert.Len(t, plan.Objects(), 4)
}

func TestPlanInstall_RecordedValuesAreTheBase(t *testing.T) {
	fc := newFakeCluster(t, operatorRecord("cli", map[string]any{"registry": "opmodel.dev=mirror", "replicas": int64(2)}))
	r := &fakeRender{objs: moduleObjects(renderOpts{})}
	file := writeValues(t, "values: replicas: 3\n")

	plan, err := PlanInstall(context.Background(), newEnv(fc, r), testResolution("v0.1.0"), defaultTarget,
		PlanOptions{ValuesFiles: []string{file}, Timeout: time.Second})
	require.NoError(t, err)
	require.NotNil(t, plan.PrevRecord)
	assert.Equal(t, "opmodel.dev=mirror", plan.Render.Values["registry"])
	assert.EqualValues(t, 3, plan.Render.Values["replicas"])
	assert.Empty(t, fc.Writes())
}

// Every refusal path returns before any write.
func TestPlanInstall_RefusalsWriteNothing(t *testing.T) {
	validation := &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: errors.New("values.bogus: field not allowed")}
	foreignNS := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"name": OperatorNamespace},
	}}
	foreignRole := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRole", "metadata": map[string]any{"name": "team-a-role"},
	}}
	cases := []struct {
		name    string
		cluster []*unstructured.Unstructured
		render  *fakeRender
		target  Target
		setup   func(*fakeCluster)
		timeout time.Duration
		want    []string
		check   func(*testing.T, error)
	}{
		{
			name:    "recorded value the target rejects",
			cluster: []*unstructured.Unstructured{operatorRecord("cli", map[string]any{"bogus": true})},
			render:  &fakeRender{err: validation},
			want:    []string{"values.bogus", "--reset-values"},
			check: func(t *testing.T, err error) {
				var exitErr *opmexit.ExitError
				require.ErrorAs(t, err, &exitErr)
				assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
			},
		},
		{
			name:   "rejected values without a record do not name --reset-values",
			render: &fakeRender{err: validation},
			check: func(t *testing.T, err error) {
				assert.NotContains(t, err.Error(), "--reset-values")
			},
		},
		{
			name:   "operator newer than the CLI",
			render: &fakeRender{objs: moduleObjects(renderOpts{imageTag: "v1.1.0"})},
			target: Target{ModuleVersion: "v0.4.0", OperatorVersion: "v1.1.0"},
			want:   []string{"upgrade the CLI"},
		},
		{
			name:   "image disagrees",
			render: &fakeRender{objs: moduleObjects(renderOpts{imageTag: "v1.0.0-beta.5"})},
			want:   []string{"tagged v1.0.0-beta.5"},
		},
		{
			name:   "CRD below the floor",
			render: &fakeRender{objs: moduleObjects(renderOpts{noCRDFloor: true})},
			want:   []string{"spec.owner and status.inventory"},
		},
		{
			name:   "status RBAC denied",
			render: &fakeRender{objs: moduleObjects(renderOpts{})},
			setup:  func(fc *fakeCluster) { fc.denyStatusRBAC = true },
			want:   []string{"moduleinstances/status"},
		},
		{
			name:    "an object exists and is not OPM's",
			cluster: []*unstructured.Unstructured{foreignRole},
			render:  &fakeRender{objs: moduleObjects(renderOpts{extraRole: "team-a-role"})},
			want:    []string{"ClusterRole/team-a-role", "not managed by OPM"},
			check: func(t *testing.T, err error) {
				var ge *GuardError
				require.ErrorAs(t, err, &ge)
			},
		},
		{
			name:    "an object of an earlier manifest fails the proof",
			cluster: []*unstructured.Unstructured{foreignNS},
			render:  &fakeRender{objs: moduleObjects(renderOpts{})},
			want:    []string{"operator migration refused", "Namespace/opm-operator-system: label app.kubernetes.io/managed-by is missing"},
			check: func(t *testing.T, err error) {
				var mr *MigrationRefusalError
				require.ErrorAs(t, err, &mr)
			},
		},
		{
			name:    "terminating object outlives the budget",
			cluster: []*unstructured.Unstructured{terminatingFixture(deploymentFixture(false))},
			render:  &fakeRender{objs: moduleObjects(renderOpts{})},
			timeout: 30 * time.Millisecond,
			want:    []string{"Deployment/opm-operator-controller-manager", "timed out after"},
		},
		{
			name:    "operator-owned record",
			cluster: []*unstructured.Unstructured{operatorRecord("operator", nil)},
			render:  &fakeRender{objs: moduleObjects(renderOpts{})},
			want:    []string{"is not spec.owner: cli", "set spec.owner to cli"},
		},
		{
			name:    "record without an owner",
			cluster: []*unstructured.Unstructured{ownerlessRecord()},
			render:  &fakeRender{objs: moduleObjects(renderOpts{})},
			want:    []string{"is not spec.owner: cli", "set spec.owner to cli"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fastPolling(t)
			fc := newFakeCluster(t, c.cluster...)
			if c.setup != nil {
				c.setup(fc)
			}
			target := c.target
			if target == (Target{}) {
				target = defaultTarget
			}
			timeout := c.timeout
			if timeout == 0 {
				timeout = time.Second
			}
			_, err := PlanInstall(context.Background(), newEnv(fc, c.render), testResolution(target.ModuleVersion), target, PlanOptions{Timeout: timeout})
			require.Error(t, err)
			for _, w := range c.want {
				assert.Contains(t, err.Error(), w)
			}
			if c.check != nil {
				c.check(t, err)
			}
			assert.Empty(t, fc.Writes(), "a refused install writes nothing")
		})
	}
}

// "Install right after uninstall waits out the terminating objects".
func TestPlanInstall_WaitsOutTerminatingObjects(t *testing.T) {
	fastPolling(t)
	doomed := terminatingFixture(deploymentFixture(false))
	fc := newFakeCluster(t, doomed)
	deleteLater(t, fc.client, doomed, 20*time.Millisecond)

	_, err := PlanInstall(context.Background(), newEnv(fc, &fakeRender{objs: moduleObjects(renderOpts{})}),
		testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: 2 * time.Second})
	require.NoError(t, err)
	assert.Equal(t, []string{"delete deployments"}, fc.Writes(), "only the test's own delete")
}

// The CRDs-only plan guards only the CRDs and skips the status check.
func TestPlanInstall_CRDsOnlyGuardsOnlyTheCRDs(t *testing.T) {
	foreignNS := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"name": OperatorNamespace},
	}}
	fc := newFakeCluster(t, foreignNS)
	fc.denyStatusRBAC = true

	plan, err := PlanInstall(context.Background(), newEnv(fc, &fakeRender{objs: moduleObjects(renderOpts{})}),
		testResolution("v0.1.0"), defaultTarget, PlanOptions{CRDsOnly: true, Timeout: time.Second})
	require.NoError(t, err)
	assert.Len(t, plan.Objects(), 4)
	assert.Empty(t, fc.Writes())
}

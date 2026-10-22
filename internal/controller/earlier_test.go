package controller

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"

	"github.com/open-platform-model/cli/internal/output"
)

// earlierDeployment is the controller Deployment of the earlier install.
func earlierDeployment() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apps/v1", "kind": "Deployment",
		"metadata": map[string]any{"name": earlierControllerDeployment, "namespace": earlierControllerNamespace},
	}}
}

// manifestCRD is a CRD of an install from the release manifest: no OPM labels.
func manifestCRD(rendered []*unstructured.Unstructured) *unstructured.Unstructured {
	crd := liveOf(rendered, kindCustomResourceDefinition, CRDNames()[0])
	crd.SetLabels(nil)
	return crd
}

// adoptable annotates a live CRD for adoption by the controller's instance.
func adoptable(crd *unstructured.Unstructured) *unstructured.Unstructured {
	crd.SetAnnotations(map[string]string{opmlabels.AnnotationAdopt: testInstanceUUID})
	return crd
}

func TestEarlierInstallNotice(t *testing.T) {
	rendered := moduleObjects(renderOpts{})
	forbidden := func(fc *fakeCluster) {
		fc.fake.PrependReactor("get", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "apps", Resource: "deployments"}, earlierControllerDeployment, errors.New("no get"))
		})
	}
	cases := []struct {
		name    string
		cluster []*unstructured.Unstructured
		setup   func(*fakeCluster)
		notice  bool
	}{
		{name: "the earlier Deployment exists", cluster: []*unstructured.Unstructured{earlierDeployment()}, notice: true},
		{name: "no Deployment", cluster: nil},
		{name: "the controller of this install only", cluster: []*unstructured.Unstructured{liveOf(rendered, kindDeployment, ControllerDeploymentName)}},
		{name: "the read is denied", cluster: []*unstructured.Unstructured{earlierDeployment()}, setup: forbidden},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fc := newFakeCluster(t, c.cluster...)
			if c.setup != nil {
				c.setup(fc)
			}
			got := EarlierInstallNotice(context.Background(), fc.client)
			if !c.notice {
				assert.Empty(t, got)
				return
			}
			assert.Contains(t, got, "Deployment opm-operator-controller-manager in namespace opm-operator-system")
			assert.Contains(t, got, `"Install over the earlier controller"`)
			for _, banned := range []string{"delete", "remove", "rename", "kubectl"} {
				assert.NotContains(t, got, banned, "the line points to the page and gives no deletion advice")
			}
			assert.Empty(t, fc.Writes())
		})
	}
}

// The hint follows a guard refusal when the earlier Deployment is there, for a
// manifest install and for an install by the earlier module, in the full form
// and in the --crds-only form. The refusal, its type and its text stay.
func TestHintEarlierInstall_OnARefusal(t *testing.T) {
	rendered := moduleObjects(renderOpts{})
	crdOfOtherInstance := func() *unstructured.Unstructured {
		return ownedBy(liveOf(rendered, kindCustomResourceDefinition, CRDNames()[0]), otherInstanceUUID)
	}
	cases := []struct {
		name     string
		crdsOnly bool
		cluster  []*unstructured.Unstructured
		hinted   bool
		refusal  string
	}{
		{name: "manifest install, full", cluster: []*unstructured.Unstructured{manifestCRD(rendered), earlierDeployment()}, hinted: true, refusal: "exists and is not managed by OPM"},
		{name: "module install, full", cluster: []*unstructured.Unstructured{crdOfOtherInstance(), earlierDeployment()}, hinted: true, refusal: "belongs to module instance " + otherInstanceUUID},
		{name: "manifest install, crds-only", crdsOnly: true, cluster: []*unstructured.Unstructured{manifestCRD(rendered), earlierDeployment()}, hinted: true, refusal: "exists and is not managed by OPM"},
		{name: "module install, crds-only", crdsOnly: true, cluster: []*unstructured.Unstructured{crdOfOtherInstance(), earlierDeployment()}, hinted: true, refusal: "belongs to module instance " + otherInstanceUUID},
		{name: "refusal and no Deployment", cluster: []*unstructured.Unstructured{manifestCRD(rendered)}, refusal: "exists and is not managed by OPM"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fastPolling(t)
			fc := newFakeCluster(t, c.cluster...)
			env := newEnv(fc, &fakeRender{objs: rendered})
			var err error
			if c.crdsOnly {
				prev := embeddedCRDs
				embeddedCRDs = func() ([]*unstructured.Unstructured, error) { return renderedCRDs(rendered), nil }
				t.Cleanup(func() { embeddedCRDs = prev })
				_, err = PlanCRDsOnly(context.Background(), env, PlanOptions{Timeout: time.Second})
			} else {
				_, err = PlanInstall(context.Background(), env, testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: time.Second})
			}
			var ge *GuardError
			require.ErrorAs(t, err, &ge)
			plain := err.Error()

			got := HintEarlierInstall(context.Background(), fc.client, err)

			require.ErrorAs(t, got, &ge, "the refusal stays in the chain, so the exit code stays 2")
			assert.Contains(t, got.Error(), c.refusal)
			assert.Contains(t, got.Error(), plain, "the refusal text is unchanged")
			var hinted *HintedError
			if !c.hinted {
				assert.NotErrorAs(t, got, &hinted)
				assert.Equal(t, plain, got.Error())
				return
			}
			require.ErrorAs(t, got, &hinted)
			assert.Equal(t, plain+"\nhint: "+earlierInstallNotice, got.Error())
			assert.Contains(t, got.Error(), "nothing was changed\nhint: ", "the hint is the last line")
			assert.Empty(t, fc.Writes())
		})
	}
}

// A failure that is not a guard refusal gets no hint, and a denied read of the
// Deployment leaves the refusal as it is.
func TestHintEarlierInstall_LeavesOtherErrorsAlone(t *testing.T) {
	fc := newFakeCluster(t, earlierDeployment())
	other := errors.New("render failed")
	assert.Same(t, other, HintEarlierInstall(context.Background(), fc.client, other))
	owned := &OwnedRecordError{}
	assert.Same(t, owned, HintEarlierInstall(context.Background(), fc.client, error(owned)))

	fc = newFakeCluster(t, earlierDeployment())
	fc.fake.PrependReactor("get", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "apps", Resource: "deployments"}, earlierControllerDeployment, errors.New("no get"))
	})
	refusal := &GuardError{Err: errors.New("exists")}
	assert.Same(t, refusal, HintEarlierInstall(context.Background(), fc.client, error(refusal)))
}

// Checks that pass, with the earlier Deployment present, give one warning and
// no error: the CRDs carry the adopt annotation, for a manifest install and for
// an install by the earlier module.
func TestWarnEarlierInstall_OnAPassedPlan(t *testing.T) {
	rendered := moduleObjects(renderOpts{})
	cases := []struct {
		name    string
		cluster []*unstructured.Unstructured
		warned  bool
	}{
		{name: "manifest install, annotated", cluster: []*unstructured.Unstructured{adoptable(manifestCRD(rendered)), earlierDeployment()}, warned: true},
		{name: "module install, annotated", cluster: []*unstructured.Unstructured{adoptable(ownedBy(liveOf(rendered, kindCustomResourceDefinition, CRDNames()[0]), otherInstanceUUID)), earlierDeployment()}, warned: true},
		{name: "no Deployment", cluster: []*unstructured.Unstructured{adoptable(manifestCRD(rendered))}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fastPolling(t)
			var logs bytes.Buffer
			output.SetLogWriter(&logs)
			t.Cleanup(func() { output.SetLogWriter(os.Stderr) })
			fc := newFakeCluster(t, c.cluster...)

			_, err := PlanInstall(context.Background(), newEnv(fc, &fakeRender{objs: rendered}),
				testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: time.Second})
			require.NoError(t, err, "the checks pass")
			assert.Empty(t, fc.Writes(), "the warning comes before the first write")
			WarnEarlierInstall(context.Background(), fc.client)

			if c.warned {
				assert.Equal(t, 1, strings.Count(logs.String(), "the cluster still runs the controller of the earlier install"))
				assert.Contains(t, logs.String(), "WARN")
			} else {
				assert.Empty(t, logs.String())
			}
		})
	}
}

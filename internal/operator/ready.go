package operator

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

// NotReadyError reports that the operator is not installed, or installed but
// not serving. Pending names the resources that failed their readiness
// predicate — the CRDs that are not Established and the controller Deployment
// that has not rolled out.
type NotReadyError struct {
	Pending []string
	// Hint is the caller-supplied consequence line: why this particular
	// command refuses to proceed without a running operator.
	Hint string
}

func (e *NotReadyError) Error() string {
	msg := fmt.Sprintf("the opm operator is not ready (%s)", strings.Join(e.Pending, ", "))
	if e.Hint != "" {
		msg += " — " + e.Hint
	}
	return msg + "; install or repair it with 'opm operator install', then retry"
}

// CheckReady reports whether the operator is installed and serving: its CRDs
// Established and its controller Deployment rolled out. It is the single-shot
// form of the readiness machinery Install waits on (0006:D35 built it for this
// reuse): a gate, not a wait, so a down operator fails fast instead of burning
// a timeout.
//
// It locates the operator by its fixed names (names.go) and reads nothing
// else: no embedded manifest, no instance record, no Namespace. Those names
// hold for every operator release since v1.0.0-alpha.18 and for the operator
// module, so a manifest install and a module install are found alike. Any read
// that fails counts that object as not ready, so a caller proceeds only on a
// positive finding.
func CheckReady(ctx context.Context, client *kubernetes.Client) error {
	pending := pendingObjects(ctx, client, fixedTargets(), DefaultPredicate)
	if len(pending) == 0 {
		return nil
	}
	return &NotReadyError{Pending: kubernetes.DescribeObjectList(pending)}
}

// fixedTargets are the objects whose liveness defines "the operator is
// serving": the CRDs in CRDNames and the controller Deployment. Supporting
// objects (RBAC, Service, Namespace) are left out: a rolled-out Deployment
// implies them, and they add noise to the refusal message.
func fixedTargets() []*unstructured.Unstructured {
	targets := make([]*unstructured.Unstructured, 0, len(CRDNames)+1)
	for _, name := range CRDNames {
		crd := &unstructured.Unstructured{}
		crd.SetAPIVersion("apiextensions.k8s.io/v1")
		crd.SetKind(kindCustomResourceDefinition)
		crd.SetName(name)
		targets = append(targets, crd)
	}
	deploy := &unstructured.Unstructured{}
	deploy.SetAPIVersion("apps/v1")
	deploy.SetKind(kindDeployment)
	deploy.SetName(ControllerDeploymentName)
	deploy.SetNamespace(OperatorNamespace)
	return append(targets, deploy)
}

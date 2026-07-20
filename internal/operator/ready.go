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
// form of the readiness machinery Install waits on (enhancement 0006 D35 built
// it for this reuse) — a gate, not a wait, so a down operator fails fast
// instead of burning a timeout.
//
// Readiness targets come from the embedded, pinned manifest: the CRD names and
// the controller Deployment's name/namespace are stable across operator
// versions, so an older or newer installed operator is still located correctly.
func CheckReady(ctx context.Context, client *kubernetes.Client) error {
	manifest, err := EmbeddedManifest()
	if err != nil {
		return fmt.Errorf("reading embedded operator manifest: %w", err)
	}

	targets := readinessTargets(manifest)
	if len(targets) == 0 {
		return fmt.Errorf("embedded operator manifest declares no CRDs or controller Deployment")
	}

	pending := pendingObjects(ctx, client, targets, DefaultPredicate)
	if len(pending) == 0 {
		return nil
	}
	return &NotReadyError{Pending: describeObjectList(pending)}
}

// readinessTargets selects the manifest objects whose liveness defines "the
// operator is serving": the CRDs and the controller Deployment. Supporting
// objects (RBAC, Service, Namespace) are excluded — their existence is implied
// by a rolled-out Deployment and adds noise to the refusal message.
func readinessTargets(manifest []*unstructured.Unstructured) []*unstructured.Unstructured {
	var targets []*unstructured.Unstructured
	for _, obj := range manifest {
		switch obj.GetKind() {
		case kindCustomResourceDefinition, kindDeployment:
			targets = append(targets, obj)
		}
	}
	return targets
}

func describeObjectList(objs []*unstructured.Unstructured) []string {
	names := make([]string, len(objs))
	for i, obj := range objs {
		if ns := obj.GetNamespace(); ns != "" {
			names[i] = fmt.Sprintf("%s/%s in %s", obj.GetKind(), obj.GetName(), ns)
		} else {
			names[i] = fmt.Sprintf("%s/%s", obj.GetKind(), obj.GetName())
		}
	}
	return names
}

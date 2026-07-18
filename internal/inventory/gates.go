package inventory

import (
	"context"
	"fmt"

	"golang.org/x/mod/semver"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

// crdGVR is the CustomResourceDefinition GroupVersionResource used by the
// CRD-presence and field-floor gates.
var crdGVR = schema.GroupVersionResource{
	Group:    "apiextensions.k8s.io",
	Version:  "v1",
	Resource: "customresourcedefinitions",
}

const (
	crdInstallHint  = "run 'opm operator install --crds-only'"
	rbacInstallHint = "grant the moduleinstances/status subresource or run 'opm operator install --crds-only --rbac'"
)

// GateCRDPresent verifies the ModuleInstance CRD exists. A NotFound is the
// actionable missing-CRD error; any other read error is surfaced as-is.
func GateCRDPresent(ctx context.Context, client *kubernetes.Client) error {
	_, err := client.Dynamic.Resource(crdGVR).Get(ctx, CRDNameModuleInstances, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("ModuleInstance CRD not found — %s", crdInstallHint)
		}
		return fmt.Errorf("checking ModuleInstance CRD: %w", err)
	}
	return nil
}

// GateCRDFieldFloor verifies the installed ModuleInstance CRD's served storage
// version schema contains spec.owner and status.inventory. This guards against
// an outdated CRD whose API server would silently prune the CLI's spec.owner
// marker or reject its status.inventory write.
func GateCRDFieldFloor(ctx context.Context, client *kubernetes.Client) error {
	crd, err := client.Dynamic.Resource(crdGVR).Get(ctx, CRDNameModuleInstances, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("reading ModuleInstance CRD: %w", err)
	}

	root, err := servedStorageSchema(crd)
	if err != nil {
		return err
	}

	if !hasSchemaProperty(root, "spec", "owner") || !hasSchemaProperty(root, "status", "inventory") {
		return fmt.Errorf("ModuleInstance CRD is missing required fields — %s", crdInstallHint)
	}
	return nil
}

// GateOperatorVersionCeiling reads the cluster-scoped Platform singleton and
// refuses when the operator's advertised version is semver-newer than the
// CLI's. Absent Platform or absent status.operatorVersion means a solo cluster
// (or a pre-A6 operator) and the gate is skipped. A dev-build CLI or an
// RBAC-denied Platform read degrades to skip-with-warning so a namespace-scoped
// user can still apply.
func GateOperatorVersionCeiling(ctx context.Context, client *kubernetes.Client, cliVersion string) error {
	normalizedCLI := ensureVPrefix(cliVersion)
	if cliVersion == "" || cliVersion == "dev" || !semver.IsValid(normalizedCLI) {
		output.Warn("skipping operator-version ceiling check: CLI version is not a released semver", "version", cliVersion)
		return nil
	}

	plat, err := client.Dynamic.Resource(PlatformGVR).Get(ctx, PlatformSingletonName, metav1.GetOptions{})
	if err != nil {
		switch {
		case apierrors.IsNotFound(err):
			output.Debug("no Platform singleton; skipping operator-version ceiling (solo cluster)")
			return nil
		case apierrors.IsForbidden(err):
			output.Warn("skipping operator-version ceiling check: reading the Platform was denied by RBAC")
			return nil
		default:
			return fmt.Errorf("reading Platform for operator-version ceiling: %w", err)
		}
	}

	//nolint:errcheck // best-effort read; wrong-typed status.operatorVersion treated as absent
	opVersion, ok, _ := unstructured.NestedString(plat.Object, "status", "operatorVersion")
	if !ok || opVersion == "" {
		output.Debug("Platform has no status.operatorVersion; skipping operator-version ceiling")
		return nil
	}

	normalizedOp := ensureVPrefix(opVersion)
	if !semver.IsValid(normalizedOp) {
		output.Warn("skipping operator-version ceiling check: operatorVersion is not valid semver", "operatorVersion", opVersion)
		return nil
	}

	if semver.Compare(normalizedOp, normalizedCLI) > 0 {
		return fmt.Errorf(
			"your CLI (%s) is older than the cluster operator (%s) — upgrade the CLI before applying against this cluster",
			cliVersion, opVersion,
		)
	}
	return nil
}

// GateStatusRBAC issues a SelfSubjectAccessReview for patching
// moduleinstances/status in the target namespace. It runs in CLI-executor mode
// only and aborts before any resource is applied when access is denied, so
// resources are never deployed without a recordable inventory.
func GateStatusRBAC(ctx context.Context, client *kubernetes.Client, namespace string) error {
	ssar := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace:   namespace,
				Verb:        "patch",
				Group:       GroupOpmodel,
				Resource:    ResourceModuleInstances,
				Subresource: "status",
			},
		},
	}

	resp, err := client.Clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("checking access to patch moduleinstances/status: %w", err)
	}

	if !resp.Status.Allowed {
		reason := resp.Status.Reason
		if reason == "" {
			reason = "access denied"
		}
		return fmt.Errorf(
			"cannot record inventory: patching moduleinstances/status is denied in namespace %q (%s) — %s",
			namespace, reason, rbacInstallHint,
		)
	}
	return nil
}

// servedStorageSchema returns the openAPIV3Schema of the CRD's storage version
// (falling back to the first served version).
func servedStorageSchema(crd *unstructured.Unstructured) (map[string]any, error) {
	//nolint:errcheck // best-effort read; a wrong-typed versions field is handled as "no versions"
	versions, ok, _ := unstructured.NestedSlice(crd.Object, "spec", "versions")
	if !ok || len(versions) == 0 {
		return nil, fmt.Errorf("ModuleInstance CRD has no versions")
	}

	var chosen map[string]any
	for _, v := range versions {
		vm, ok := v.(map[string]any)
		if !ok {
			continue
		}
		if storage, _, _ := unstructured.NestedBool(vm, "storage"); storage { //nolint:errcheck // best-effort; missing → false
			chosen = vm
			break
		}
		if served, _, _ := unstructured.NestedBool(vm, "served"); served && chosen == nil { //nolint:errcheck // best-effort; missing → false
			chosen = vm
		}
	}
	if chosen == nil {
		return nil, fmt.Errorf("ModuleInstance CRD has no served or storage version")
	}

	//nolint:errcheck // best-effort read; a missing schema is reported below
	root, ok, _ := unstructured.NestedMap(chosen, "schema", "openAPIV3Schema")
	if !ok {
		return nil, fmt.Errorf("ModuleInstance CRD storage version has no schema")
	}
	return root, nil
}

// hasSchemaProperty reports whether openAPIV3Schema.properties.<parent>.properties.<child>
// exists as an object.
func hasSchemaProperty(root map[string]any, parent, child string) bool {
	_, ok, _ := unstructured.NestedMap(root, "properties", parent, "properties", child) //nolint:errcheck // best-effort; wrong-typed → not present
	return ok
}

func ensureVPrefix(v string) string {
	if v == "" || v[0] == 'v' {
		return v
	}
	return "v" + v
}

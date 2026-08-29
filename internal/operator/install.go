package operator

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

// InstallOptions configures an install run.
type InstallOptions struct {
	// CRDsOnly applies only the CustomResourceDefinition documents and waits
	// only for their Established condition.
	CRDsOnly bool

	// Version, when non-empty, fetches install.yaml from that opm-operator
	// release tag instead of using the embedded artifact.
	Version string

	// Timeout bounds the terminating guard and the readiness wait together.
	Timeout time.Duration

	// RBAC configures optional opm-cli-user ClusterRole/ClusterRoleBinding
	// emission, appended to the plan regardless of CRDsOnly.
	RBAC RBACOptions
}

// InstallResult reports the outcome of an install run.
type InstallResult struct {
	// Version is the opm-operator version that was installed: the embedded
	// pin, or the fetched --version tag.
	Version string

	// Source is "embedded" or "fetched".
	Source string

	// Applied is the number of resources server-side-applied.
	Applied int
}

// Install waits for any planned object still terminating on the cluster to
// disappear, then server-side-applies the operator manifest (or just its CRD
// subset) with field manager opm-cli, then waits for the applied resources to
// become ready. Both waits share the single opts.Timeout budget. Apply stops
// at the first resource error — a partially applied operator install is not a
// state worth waiting on.
func Install(ctx context.Context, client *kubernetes.Client, opts InstallOptions) (*InstallResult, error) {
	manifest, version, source, err := resolveManifest(ctx, opts.Version)
	if err != nil {
		return nil, err
	}

	plan := InstallPlan(manifest)
	if opts.CRDsOnly {
		plan = CRDsOnlyPlan(manifest)
	}
	if rbacObjs := opts.RBAC.Objects(); len(rbacObjs) > 0 {
		plan = append(plan, rbacObjs...)
		sortByWeightAscending(plan)
	}

	result := &InstallResult{Version: version, Source: source}

	// One budget for the terminating guard and the readiness wait: the user
	// reasons about a single --timeout per command.
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	budgetStart := time.Now()

	if err := waitForTerminating(ctx, client, plan, budgetStart); err != nil {
		return result, err
	}

	for _, obj := range plan {
		status, err := kubernetes.ApplyOne(ctx, client, obj, kubernetes.ApplyOptions{})
		if err != nil {
			return result, fmt.Errorf("applying %s/%s: %w", obj.GetKind(), obj.GetName(), err)
		}
		result.Applied++
		output.Info(output.FormatResourceLine(obj.GetKind(), obj.GetNamespace(), obj.GetName(), status))
	}

	if err := Wait(ctx, client, plan, DefaultPredicate, budgetStart); err != nil {
		return result, err
	}

	return result, nil
}

// waitForTerminating is the pre-apply guard: it reads every planned object
// and waits for those that exist with a deletionTimestamp (typically left by
// a previous uninstall's foreground delete) to disappear, under the caller's
// ctx deadline. Applying onto a terminating object succeeds and is then undone
// by the garbage collector, so the install would wait out its timeout on a
// resource that no longer exists. Absent and live objects never delay apply.
func waitForTerminating(ctx context.Context, client *kubernetes.Client, plan []*unstructured.Unstructured, since time.Time) error {
	terminating, err := terminatingObjects(ctx, client, plan)
	if err != nil {
		return err
	}
	if len(terminating) == 0 {
		return nil
	}

	for _, obj := range terminating {
		output.Info(output.FormatResourceLine(obj.GetKind(), obj.GetNamespace(), obj.GetName(), "waiting to finish terminating"))
	}
	return WaitAbsent(ctx, client, terminating, since)
}

// terminatingObjects returns the planned objects that exist on the cluster
// with metadata.deletionTimestamp set. A NotFound read means nothing to wait
// on; any other read error is returned, since the guard cannot tell whether
// the object is terminating.
func terminatingObjects(ctx context.Context, client *kubernetes.Client, plan []*unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	var terminating []*unstructured.Unstructured
	for _, obj := range plan {
		live, err := client.ResourceClient(kubernetes.GVRFromUnstructured(obj), obj.GetNamespace()).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("checking %s/%s before apply: %w", obj.GetKind(), obj.GetName(), err)
		}
		if live.GetDeletionTimestamp() != nil {
			terminating = append(terminating, obj)
		}
	}
	return terminating, nil
}

// resolveManifest returns the manifest to install: the embedded, pinned
// artifact by default, or a fetched one when version is non-empty.
func resolveManifest(ctx context.Context, version string) (objs []*unstructured.Unstructured, resolvedVersion, source string, err error) {
	return resolveManifestFrom(ctx, operatorReleaseBaseURL, version)
}

// resolveManifestFrom is the testable core of resolveManifest — baseURL is
// injectable so tests can point a --version fetch at a stub server.
func resolveManifestFrom(ctx context.Context, baseURL, version string) (objs []*unstructured.Unstructured, resolvedVersion, source string, err error) {
	if version == "" {
		objs, err := EmbeddedManifest()
		return objs, PinnedOperatorVersion, "embedded", err
	}

	data, err := fetchManifest(ctx, baseURL, version)
	if err != nil {
		return nil, "", "", err
	}

	objs, err = ParseManifest(data)
	return objs, version, "fetched", err
}

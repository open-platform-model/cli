package operator

import (
	"context"
	"fmt"
	"time"

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

	// Timeout bounds the readiness wait.
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

// Install server-side-applies the operator manifest (or just its CRD subset)
// with field manager opm-cli, then waits, bounded by opts.Timeout, for the
// applied resources to become ready. Apply stops at the first resource error —
// a partially applied operator install is not a state worth waiting on.
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

	for _, obj := range plan {
		status, err := kubernetes.ApplyOne(ctx, client, obj, kubernetes.ApplyOptions{})
		if err != nil {
			return result, fmt.Errorf("applying %s/%s: %w", obj.GetKind(), obj.GetName(), err)
		}
		result.Applied++
		output.Info(output.FormatResourceLine(obj.GetKind(), obj.GetNamespace(), obj.GetName(), status))
	}

	if err := Wait(ctx, client, plan, DefaultPredicate, opts.Timeout); err != nil {
		return result, err
	}

	return result, nil
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

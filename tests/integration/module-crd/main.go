//go:build ignore

// Module-to-CRD integration test.
//
// Builds a CustomResourceDefinition from an OPM module via pkg/k8sgen and
// submits it to the kind-opm-dev cluster with server-side dry-run apply.
// The API server validates the CRD's openAPIV3Schema as a structural schema;
// a failure here means pkg/k8sgen emitted something Kubernetes does not
// accept (missing type, forbidden fields at root, etc.) even though
// pkg/k8sgen's unit tests pass. This is the only place in the test suite
// where we find out.
package main

import (
	"context"
	"fmt"
	"os"

	"cuelang.org/go/cue/cuecontext"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/kubernetes"
	"github.com/opmodel/cli/pkg/k8sgen"
	"github.com/opmodel/cli/pkg/loader"
)

const (
	kindContext = "kind-opm-dev"
	fixtureDir  = "tests/fixtures/valid/simple-module"
	crdGroup    = "module.opmodel.dev"
	releaseName = "opm-module-crd-test"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== OPM Module CRD Integration Test ===")
	fmt.Println()

	fmt.Printf("1. Loading module from %s...\n", fixtureDir)
	cueCtx := cuecontext.New()
	modVal, err := loader.LoadModulePackage(cueCtx, fixtureDir)
	if err != nil {
		die("loading module: %v", err)
	}
	fmt.Println("   OK")

	fmt.Println("2. Building CRD from #config...")
	crdManifest, err := k8sgen.BuildCRD(modVal, k8sgen.Options{Group: crdGroup})
	if err != nil {
		die("building CRD: %v", err)
	}
	fmt.Printf("   OK: %s/%s\n", crdManifest.GetKind(), crdManifest.GetName())

	fmt.Printf("3. Creating Kubernetes client (context: %s)...\n", kindContext)
	client, err := kubernetes.NewClient(kubernetes.ClientOptions{
		Context: kindContext,
	})
	if err != nil {
		die("creating k8s client: %v", err)
	}
	fmt.Println("   OK")

	fmt.Println("4. Server-side dry-run apply (validates structural schema)...")
	result, err := kubernetes.Apply(
		ctx,
		client,
		[]*unstructured.Unstructured{crdManifest},
		releaseName,
		kubernetes.ApplyOptions{DryRun: true},
	)
	if err != nil {
		die("apply: %v", err)
	}
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "   reject: %s/%s: %v\n", e.Kind, e.Name, e.Err)
		}
		die("server rejected CRD (%d error(s))", len(result.Errors))
	}
	fmt.Printf("   OK: applied=%d created=%d configured=%d unchanged=%d\n",
		result.Applied, result.Created, result.Configured, result.Unchanged)

	fmt.Println()
	fmt.Println("=== PASS ===")
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
	os.Exit(1)
}

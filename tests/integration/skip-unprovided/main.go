//go:build ignore

// Integration test for `opm instance apply --skip-unprovided`.
//
// Applies the skip-unprovided test module's instance package
// (internal/workflow/render/testdata/skip-unprovided/instance), whose "db"
// component attaches the provider-fulfilled backup trait while its backup
// value is true, against the cluster Platform of kind-opm-dev, which carries
// no backup provider.
//
// Scenarios:
//   - Default apply: refused with exit 2, the hint naming a provider,
//     --platform <dir> and --skip-unprovided; nothing is written.
//   - Apply with --skip-unprovided: succeeds, warns about the skipped trait
//     and stamps module-instance.opmodel.dev/skipped-contracts on the
//     ModuleInstance.
//   - Re-apply with backup false: nothing is skipped, and the annotation is
//     removed.
//
// Requires:
//   - kind cluster at context "kind-opm-dev" with the ModuleInstance CRD and
//     a Platform named "cluster" that carries no backup provider
//     (`kubectl get transformerregistrations -A` lists none)
//   - OPM_REGISTRY routing opmodel.dev to a registry serving core and the
//     catalogs
//
// Run with: go run tests/integration/skip-unprovided/main.go
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
)

const (
	clusterContext = "kind-opm-dev"
	testNamespace  = "opm-skip-unprovided-itest"

	// instanceName is the metadata.name the fixture's instance package
	// declares.
	instanceName    = "backup-demo"
	instanceFixture = "internal/workflow/render/testdata/skip-unprovided/instance"

	skippedAnnotation = "module-instance.opmodel.dev/skipped-contracts"
	backupFQN         = "opmodel.dev/catalogs/opm/traits/backup@v1alpha1"

	binaryPath = "bin/opm-skip-unprovided-itest"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== OPM Skip Unprovided Integration Test ===")
	fmt.Println()

	buildBinary()
	seedHome()
	defer os.RemoveAll(itestHome)

	client, err := kubernetes.NewClient(kubernetes.ClientOptions{Context: clusterContext})
	check("creating Kubernetes client", err)

	cleanup(ctx, client)
	_, err = client.EnsureNamespace(ctx, testNamespace, false)
	check("ensuring test namespace", err)

	step(1, "Default apply refuses the unprovided backup trait")
	stdout, stderr, exitCode := runInstanceApply()
	if exitCode != 2 {
		failf("default apply exited %d, want 2:\n%s\n%s", exitCode, stdout, stderr)
	}
	for _, want := range []string{"platform: cluster Platform", backupFQN, "Hint: a provider-fulfilled contract has no provider on this platform", "--platform <dir>", "--skip-unprovided"} {
		if !strings.Contains(stderr, want) {
			failf("default apply output lacks %q:\n%s", want, stderr)
		}
	}
	if rec, err := inventory.GetRecord(ctx, client, instanceName, testNamespace); err != nil || rec != nil {
		failf("a refused apply must write no ModuleInstance (record %v, err %v)", rec, err)
	}
	fmt.Println("   OK: refused with exit 2 and the unprovided hint; nothing written")

	step(2, "Apply with --skip-unprovided records the skipped contract")
	stdout, stderr, exitCode = runInstanceApply("--skip-unprovided")
	if exitCode != 0 {
		failf("apply --skip-unprovided exited %d:\n%s\n%s", exitCode, stdout, stderr)
	}
	warning := fmt.Sprintf("component %q: skipped provider-fulfilled trait %q (no provider on this platform)", "db", backupFQN)
	if !strings.Contains(stderr, warning) {
		failf("apply --skip-unprovided did not warn %q:\n%s", warning, stderr)
	}
	want := "db=" + backupFQN
	if got := annotation(ctx, client); got != want {
		failf("annotation %s = %q, want %q", skippedAnnotation, got, want)
	}
	fmt.Printf("   OK: %s: %s\n", skippedAnnotation, want)

	step(3, "Re-apply with backup false clears the annotation")
	valuesDir, err := os.MkdirTemp("", "opm-skip-unprovided-values-*")
	check("creating values dir", err)
	defer os.RemoveAll(valuesDir)
	valuesFile := filepath.Join(valuesDir, "values_backup_off.cue")
	check("writing values file", os.WriteFile(valuesFile, []byte("values: backup: false\n"), 0o600))
	stdout, stderr, exitCode = runInstanceApply("-f", valuesFile)
	if exitCode != 0 {
		failf("re-apply with backup false exited %d:\n%s\n%s", exitCode, stdout, stderr)
	}
	if strings.Contains(stderr, "skipped provider-fulfilled") {
		failf("a render with nothing to skip warned about a skip:\n%s", stderr)
	}
	if got := annotation(ctx, client); got != "" {
		failf("annotation %s = %q after a complete apply, want absent", skippedAnnotation, got)
	}
	fmt.Println("   OK: annotation removed")

	cleanup(ctx, client)
	fmt.Println()
	fmt.Println("=== All skip-unprovided scenarios passed ===")
}

func step(n int, title string) {
	fmt.Println()
	fmt.Printf("--- Step %d: %s\n", n, title)
}

func check(what string, err error) {
	if err != nil {
		failf("%s: %v", what, err)
	}
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
	os.Exit(1)
}

func buildBinary() {
	fmt.Println("Building opm binary for integration test...")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/opm")
	out, err := cmd.CombinedOutput()
	if err != nil {
		failf("building opm: %v\n%s", err, out)
	}
}

// itestHome is the temp HOME seeded with the config `opm config init`
// writes, so the binary never reads or writes the invoking user's ~/.opm.
var itestHome string

// realKubeconfig is the invoking user's kubeconfig path, resolved before the
// binary's HOME is redirected to the temp dir.
var realKubeconfig string

func seedHome() {
	if home, err := os.UserHomeDir(); err == nil {
		realKubeconfig = filepath.Join(home, ".kube", "config")
	}
	dir, err := os.MkdirTemp("", "opm-skip-unprovided-itest-home-*")
	check("creating temp HOME", err)
	opmDir := filepath.Join(dir, ".opm")
	check("creating temp ~/.opm", os.MkdirAll(opmDir, 0o700))
	check("writing temp config.cue", os.WriteFile(filepath.Join(opmDir, "config.cue"), []byte(config.DefaultConfigTemplate), 0o600))
	itestHome = dir
}

// runInstanceApply runs `opm instance apply` on the fixture's instance
// package in the test namespace, with extra flags appended.
func runInstanceApply(extra ...string) (stdout, stderr string, exitCode int) {
	args := append([]string{"instance", "apply", instanceFixture, "--context", clusterContext, "-n", testNamespace}, extra...)
	cmd := exec.Command("./"+binaryPath, args...)
	cmd.Env = append(os.Environ(),
		"OPM_REGISTRY="+os.Getenv("OPM_REGISTRY"),
		"HOME="+itestHome,
		"OPM_KUBECONFIG="+realKubeconfig,
	)
	if cache, err := os.UserCacheDir(); err == nil && os.Getenv("CUE_CACHE_DIR") == "" {
		cmd.Env = append(cmd.Env, "CUE_CACHE_DIR="+filepath.Join(cache, "cue"))
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		failf("running opm: %v", err)
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// annotation reads the skipped-contracts annotation off the live
// ModuleInstance, "" when absent.
func annotation(ctx context.Context, client *kubernetes.Client) string {
	obj, err := client.ResourceClient(inventory.ModuleInstanceGVR, testNamespace).Get(ctx, instanceName, metav1.GetOptions{})
	check("reading the ModuleInstance", err)
	return obj.GetAnnotations()[skippedAnnotation]
}

func cleanup(ctx context.Context, client *kubernetes.Client) {
	records, err := inventory.ListRecords(ctx, client, testNamespace)
	if err == nil {
		for _, r := range records {
			_ = inventory.DeleteCR(ctx, client, r.Name, r.Namespace)
		}
	}
	_ = client.Clientset.CoreV1().Namespaces().Delete(ctx, testNamespace, metav1.DeleteOptions{})
	deadline := time.Now().Add(60 * time.Second)
	for {
		_, err := client.Clientset.CoreV1().Namespaces().Get(ctx, testNamespace, metav1.GetOptions{})
		if apierrors.IsNotFound(err) || time.Now().After(deadline) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

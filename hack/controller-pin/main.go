// Command controller-pin writes and checks the cli's controller module pin,
// internal/controller/pin.go: the controller module version `opm controller
// install` installs by default and the controller release that version
// deploys, read from the module's own controller package without a render.
// It also renders that version as install renders the controller's instance
// and writes the render's CRDs to internal/controller/crds/, the manifests
// `opm controller install --crds-only` applies without a registry. It is not
// linked into the opm binary.
//
//	go run ./hack/controller-pin <version>        # task controller:pin VERSION=<version>
//	go run ./hack/controller-pin --check          # G1: the pin is served and consistent
//	go run ./hack/controller-pin --crds-from <module-dir>
//	go run ./hack/controller-pin select <max> <cli-version>
//
// --crds-from writes only the CRD manifests, from a local checkout of the
// controller module, for the window before that version is published; the
// re-pin against the published version must then change nothing.
//
// select prints the newest published module version at or below <max>
// (bare SemVer, the cascade resolver's answer) whose controller MAJOR.MINOR is
// not above <cli-version>'s, the rule install applies to a target, and exits
// 3 when no version qualifies.
//
// The registry mapping is OPM_REGISTRY, else CUE_REGISTRY, else the cli's
// default mapping.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/controller"
	"github.com/open-platform-model/cli/internal/controller/crds"
	"github.com/open-platform-model/cli/internal/modref"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/publish"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
)

// pinFile is where the pin lives, relative to the repository root (the
// directory task runs in).
const pinFile = "internal/controller/pin.go"

// crdDir is where the CRD manifests live, relative to the repository root.
const crdDir = "internal/controller/crds"

// controllerConst is the constant the pin records the controller version in.
const controllerConst = "PinnedControllerVersion"

// exitNone is select's "no version qualifies", the cascade's stay code.
const exitNone = 3

func main() {
	args := os.Args[1:]
	if len(args) == 1 && args[0] == "--check" {
		// G1 reads every stderr line as a finding; the render's progress
		// lines are not findings.
		output.SetLogWriter(io.Discard)
	}
	os.Exit(run(context.Background(), args, os.Stdout, os.Stderr, paths{
		mapping: registryMapping(),
		pin:     pinFile,
		crds:    crdDir,
		render:  renderModule,
	}))
}

// paths is what one run works on: the registry mapping, the pin file, the
// manifest directory and the render (tests replace it).
type paths struct {
	mapping string
	pin     string
	crds    string
	render  renderFunc
}

// renderFunc renders the controller's instance from a published module
// version, or from the local module directory dir when res is nil, and
// returns the rendered objects.
type renderFunc func(ctx context.Context, mapping string, res *modref.Resolution, dir string) ([]*unstructured.Unstructured, error)

// renderModule renders the controller's instance as a fresh install does: no
// recorded and no file values, through controller.ModuleRenderOpts.
func renderModule(ctx context.Context, mapping string, res *modref.Resolution, dir string) ([]*unstructured.Unstructured, error) {
	values, err := controller.MergeValues(controller.ValuesInput{})
	if err != nil {
		return nil, err
	}
	cfg := &config.GlobalConfig{Registry: mapping}
	k8s := &config.ResolvedKubernetesConfig{Namespace: config.ResolvedField{Value: controller.ControllerNamespace}}
	opts := controller.ModuleRenderOpts(cfg, k8s, res, values.Source)
	if res == nil {
		opts.ModulePath = dir
	}
	result, err := workflowrender.FromModule(ctx, opts)
	if err != nil {
		return nil, err
	}
	return result.Resources, nil
}

func registryMapping() string {
	for _, key := range []string{"OPM_REGISTRY", "CUE_REGISTRY"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return config.DefaultRegistry
}

// registry is what the pin tool reads from the registry.
type registry interface {
	modref.Source
	controller.ModuleFetcher
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, p paths) int {
	fail := func(err error) int {
		fmt.Fprintln(stderr, "controller-pin:", err)
		return 1
	}
	if len(args) == 2 && args[0] == "--crds-from" {
		return runCRDsFrom(ctx, stdout, stderr, p, args[1])
	}
	src, err := modref.NewSource(p.mapping)
	if err != nil {
		return fail(err)
	}
	reg, ok := src.(registry)
	if !ok {
		return fail(errors.New("the registry client cannot fetch module sources"))
	}

	switch {
	case len(args) == 1 && args[0] == "--check":
		return runCheck(ctx, reg, stdout, stderr, p)
	case len(args) == 3 && args[0] == "select":
		return runSelect(ctx, reg, stdout, stderr, args[1], args[2])
	case len(args) == 1 && !strings.HasPrefix(args[0], "-"):
		return runPin(ctx, reg, stdout, stderr, p, args[0])
	default:
		return fail(errors.New("usage: controller-pin <version> | --check | --crds-from <module-dir> | select <max-version> <cli-version>"))
	}
}

func runCheck(ctx context.Context, reg registry, stdout, stderr io.Writer, p paths) int {
	problems, err := check(ctx, reg, p)
	if err != nil {
		fmt.Fprintln(stderr, "controller-pin:", err)
		return 1
	}
	for _, p := range problems {
		fmt.Fprintln(stderr, "controller-pin:", p)
	}
	if len(problems) > 0 {
		return 1
	}
	fmt.Fprintf(stdout, "%s: pin is served and consistent, and %s holds the CRDs it renders\n", p.pin, p.crds)
	return 0
}

func runSelect(ctx context.Context, reg registry, stdout, stderr io.Writer, maxVersion, cliVersion string) int {
	v, err := selectVersion(ctx, reg, maxVersion, cliVersion)
	if err != nil {
		fmt.Fprintln(stderr, "controller-pin:", err)
		return 1
	}
	if v == "" {
		fmt.Fprintf(stderr, "controller-pin: no published %s version at or below %s deploys a controller the cli %s can drive\n",
			controller.ControllerModulePath, maxVersion, cliVersion)
		return exitNone
	}
	fmt.Fprintln(stdout, v)
	return 0
}

// runPin writes the pin and the CRD manifests of one module version. The
// render runs before anything is written, so a version that cannot be
// resolved, read or rendered leaves both as they were.
func runPin(ctx context.Context, reg registry, stdout, stderr io.Writer, p paths, version string) int {
	err := func() error {
		res, moduleVersion, ctrlVersion, err := resolve(ctx, reg, p.mapping, version)
		if err != nil {
			return err
		}
		files, err := renderManifests(ctx, p, res, "")
		if err != nil {
			return fmt.Errorf("rendering %s %s: %w", controller.ControllerModulePath, moduleVersion, err)
		}
		if err := writeManifests(p.crds, files); err != nil {
			return err
		}
		if err := writePin(p.pin, moduleVersion, ctrlVersion); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "pinned %s %s (deploys opm-controller %s; %d CRDs in %s)\n",
			controller.ControllerModulePath, moduleVersion, ctrlVersion, len(files), p.crds)
		return nil
	}()
	if err != nil {
		fmt.Fprintln(stderr, "controller-pin:", err)
		return 1
	}
	return 0
}

// runCRDsFrom writes only the CRD manifests, from a local module directory.
func runCRDsFrom(ctx context.Context, stdout, stderr io.Writer, p paths, dir string) int {
	files, err := renderManifests(ctx, p, nil, dir)
	if err == nil {
		err = writeManifests(p.crds, files)
	}
	if err != nil {
		fmt.Fprintf(stderr, "controller-pin: rendering %s: %v\n", dir, err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %d CRDs from %s to %s; %s is unchanged\n", len(files), dir, p.crds, p.pin)
	return 0
}

// resolve resolves an exact module version (bare or "v"-prefixed) through
// the registry and reads the controller version it deploys. It returns the
// resolution, the module version bare and the controller version
// "v"-prefixed.
func resolve(ctx context.Context, reg registry, mapping, version string) (res *modref.Resolution, moduleVersion, ctrlVersion string, err error) {
	sel, err := modref.ParseSelector(strings.TrimPrefix(version, "v"))
	if err != nil || sel.Exact == "" {
		return nil, "", "", fmt.Errorf("%q is not a module version such as 0.1.0", version)
	}
	res, err = modref.Resolve(ctx, reg, modref.Request{Path: controller.ControllerModulePath, Selector: sel, Registry: mapping})
	if err != nil {
		return nil, "", "", err
	}
	ctrlVersion, err = controller.ReadControllerVersion(ctx, reg, res.Version)
	if err != nil {
		return nil, "", "", err
	}
	return res, strings.TrimPrefix(res.Version, "v"), ctrlVersion, nil
}

// renderManifests renders the controller's instance and returns its CRD
// manifests by file name.
func renderManifests(ctx context.Context, p paths, res *modref.Resolution, dir string) (map[string][]byte, error) {
	objs, err := p.render(ctx, p.mapping, res, dir)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for _, obj := range objs {
		if obj.GetKind() != crds.Kind {
			continue
		}
		data, err := crds.Manifest(obj)
		if err != nil {
			return nil, err
		}
		name := crds.FileName(obj)
		if _, dup := files[name]; dup {
			return nil, fmt.Errorf("the render holds CRD %s twice", obj.GetName())
		}
		files[name] = data
	}
	if len(files) == 0 {
		return nil, errors.New("the render holds no CustomResourceDefinition")
	}
	return files, nil
}

// writeManifests replaces every *.yaml manifest in dir with files.
func writeManifests(dir string, files map[string][]byte) error {
	old, err := readManifests(dir)
	if err != nil {
		return err
	}
	for name := range old {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil { //nolint:gosec // a source file
			return err
		}
	}
	return nil
}

// readManifests returns the *.yaml files in dir by name.
func readManifests(dir string) (map[string][]byte, error) {
	names, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for _, n := range names {
		data, err := os.ReadFile(n)
		if err != nil {
			return nil, err
		}
		files[filepath.Base(n)] = data
	}
	return files, nil
}

// manifestProblems compares the manifests on disk with the ones the pinned
// version renders.
func manifestProblems(dir, moduleVersion string, have, want map[string][]byte) []string {
	fix := fmt.Sprintf("run 'task controller:pin VERSION=%s'", moduleVersion)
	var problems []string
	for name, data := range want {
		got, ok := have[name]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("%s: missing; %s %s renders CRD %s; %s",
				filepath.Join(dir, name), controller.ControllerModulePath, moduleVersion, strings.TrimSuffix(name, ".yaml"), fix))
		case !bytes.Equal(got, data):
			problems = append(problems, fmt.Sprintf("%s: differs from the CRD %s %s renders; %s",
				filepath.Join(dir, name), controller.ControllerModulePath, moduleVersion, fix))
		}
	}
	for name := range have {
		if _, ok := want[name]; !ok {
			problems = append(problems, fmt.Sprintf("%s: %s %s renders no such CRD; %s",
				filepath.Join(dir, name), controller.ControllerModulePath, moduleVersion, fix))
		}
	}
	sort.Strings(problems)
	return problems
}

// renderPin is the pin file's content.
func renderPin(moduleVersion, ctrlVersion string) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, `// Code generated by task controller:pin; DO NOT EDIT.

package controller

// PinnedModuleVersion is the controller module version (ControllerModulePath)
// 'opm controller install' installs when --version is not given. Refresh it
// only with 'task controller:pin VERSION=<version>'.
const PinnedModuleVersion = %q

// %s is the controller release PinnedModuleVersion deploys, as the
// module's controller package states it, "v"-prefixed.
const %s = %q
`, moduleVersion, controllerConst, controllerConst, ctrlVersion)
	return format.Source(b.Bytes())
}

func writePin(file, moduleVersion, ctrlVersion string) error {
	data, err := renderPin(moduleVersion, ctrlVersion)
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o644) //nolint:gosec // a source file
}

var (
	moduleConstRE     = regexp.MustCompile(`(?m)^const PinnedModuleVersion = "([^"]*)"$`)
	controllerConstRE = regexp.MustCompile(`(?m)^const ` + controllerConst + ` = "([^"]*)"$`)
)

// readPin returns the two pinned values from the pin file.
func readPin(file string) (moduleVersion, ctrlVersion string, err error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", "", fmt.Errorf("reading the pin: %w", err)
	}
	m := moduleConstRE.FindSubmatch(data)
	o := controllerConstRE.FindSubmatch(data)
	if m == nil || o == nil {
		return "", "", fmt.Errorf("%s does not declare PinnedModuleVersion and %s", file, controllerConst)
	}
	return string(m[1]), string(o[1]), nil
}

// check re-resolves the pinned module version and reports every way the pin
// disagrees with what the registry serves. An error is a failure to check
// (an unreadable pin, a registry that cannot be reached), never a pass.
func check(ctx context.Context, reg registry, p paths) ([]string, error) {
	file := p.pin
	moduleVersion, pinnedCtrl, err := readPin(file)
	if err != nil {
		return nil, err
	}
	res, _, ctrlVersion, err := resolve(ctx, reg, p.mapping, moduleVersion)
	if err != nil {
		var refusal *modref.RefusalError
		if errors.As(err, &refusal) {
			return []string{fmt.Sprintf("%s: %s %s is not served by the registry (%s)",
				file, controller.ControllerModulePath, moduleVersion, refusal.Refusal.Headline)}, nil
		}
		var opErr *controller.VersionError
		if errors.As(err, &opErr) {
			return []string{fmt.Sprintf("%s: %v", file, err)}, nil
		}
		return nil, fmt.Errorf("looking up %s %s: %w", controller.ControllerModulePath, moduleVersion, err)
	}
	var problems []string
	if ctrlVersion != pinnedCtrl {
		problems = append(problems, fmt.Sprintf("%s: %s is %s, but %s %s deploys opm-controller %s; run 'task controller:pin VERSION=%s'",
			file, controllerConst, pinnedCtrl, controller.ControllerModulePath, moduleVersion, ctrlVersion, moduleVersion))
	}
	want, err := renderManifests(ctx, p, res, "")
	if err != nil {
		return nil, fmt.Errorf("rendering %s %s: %w", controller.ControllerModulePath, moduleVersion, err)
	}
	have, err := readManifests(p.crds)
	if err != nil {
		return nil, err
	}
	return append(problems, manifestProblems(p.crds, moduleVersion, have, want)...), nil
}

// selectVersion walks the published module versions in max's major, at or
// below max, newest first, and returns the first (bare) whose controller
// MAJOR.MINOR is not above cliVersion's. Development builds are skipped.
func selectVersion(ctx context.Context, reg registry, maxVersion, cliVersion string) (string, error) {
	maxV := "v" + strings.TrimPrefix(maxVersion, "v")
	cli := "v" + strings.TrimPrefix(cliVersion, "v")
	if !semver.IsValid(maxV) || !semver.IsValid(cli) {
		return "", fmt.Errorf("select needs two SemVer versions, got %q and %q", maxVersion, cliVersion)
	}
	versions, err := reg.ModuleVersions(ctx, controller.ControllerModulePath)
	if err != nil {
		return "", fmt.Errorf("listing %s versions: %w", controller.ControllerModulePath, err)
	}
	semver.Sort(versions)
	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		if semver.Compare(v, maxV) > 0 || semver.Major(v) != semver.Major(maxV) || publish.IsDevTag(v) {
			continue
		}
		op, err := controller.ReadControllerVersion(ctx, reg, v)
		if err != nil {
			return "", err
		}
		if semver.Compare(semver.MajorMinor(op), semver.MajorMinor(cli)) <= 0 {
			return strings.TrimPrefix(v, "v"), nil
		}
	}
	return "", nil
}

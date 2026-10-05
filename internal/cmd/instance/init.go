package instance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cuelang.org/go/mod/module"
	"github.com/open-platform-model/library/opm/kernel"
	libmodule "github.com/open-platform-model/library/opm/module"
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/cuemod"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/instinit"
	"github.com/open-platform-model/cli/internal/modref"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/publish"
	"github.com/open-platform-model/cli/pkg/loader"
)

// corePath is the module path, without major, of the OPM core schema.
const corePath = "opmodel.dev/core"

// nameRule is core's #NameType: a lowercase RFC 1123 label.
var nameRule = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

const nameRuleText = "lowercase letters, digits and '-', starting and ending with a letter or digit, at most 63 characters"

// initFlags holds the flags of `opm instance init`.
type initFlags struct {
	from, version, namespace, dir, modulePath string
}

// initInputs is one invocation after arguments, prompts and offline checks.
type initInputs struct {
	name, namespace, dir, packageModulePath string
	path                                    string // major-free module path
	selector                                modref.Selector
}

// NewInstanceInitCmd creates the instance init command (0016:D5).
func NewInstanceInitCmd(cfg *config.GlobalConfig) *cobra.Command {
	var flags initFlags

	c := &cobra.Command{
		Use:   "init [instance-name] [module-path]",
		Short: "Create an instance package for a published module",
		Long: `Create a standalone instance package that deploys a published module.

The package is a directory holding three files:

  cue.mod/module.cue  pins the module at the resolved version, core at the
                      version the module declares, and their dependencies
  instance.cue        binds the module to a #ModuleInstance with the name
                      and namespace
  values.cue          starting values: the module's initValues, else its
                      debugValues when fully concrete, else empty

The module path carries no major. Without --version the newest release of the
highest major built on this CLI's core major is chosen; --version vN floats
within major N and --version X.Y.Z pins that release.

The target directory must not exist and must not sit inside another CUE
module. The package is written completely or not at all. Init does not
validate the values; run 'opm instance vet <dir>/instance.cue' next.

Missing arguments are prompted for on a terminal, in the order instance name,
module path, namespace.

Exit codes: 0 written, 2 refused, 3 registry unreachable.

Examples:
  # Deploy cert_manager into its own namespace
  opm instance init cert-manager opmodel.dev/modules/cert_manager -n cert-manager

  # Pin a release and choose the directory
  opm instance init web opmodel.dev/modules/web_app --version 1.0.4 -n demo --dir deploy/web

  # Name the module with --from
  opm instance init web --from opmodel.dev/modules/web_app -n demo`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			return runInstanceInit(c, cfg, args, flags)
		},
	}

	c.Flags().StringVar(&flags.from, "from", "", "Module path to deploy (instead of the positional)")
	c.Flags().StringVar(&flags.version, "version", "", "vN floats within major N; X.Y.Z pins a release")
	c.Flags().StringVarP(&flags.namespace, "namespace", "n", "", "Instance namespace (prompted on a terminal)")
	c.Flags().StringVar(&flags.dir, "dir", "", "Target directory (defaults to the instance name)")
	c.Flags().StringVar(&flags.modulePath, "module-path", "", "The package's own module path (defaults to instance.local/<name>@v0)")

	return c
}

func runInstanceInit(c *cobra.Command, cfg *config.GlobalConfig, args []string, flags initFlags) error {
	in, err := collectInputs(c, args, flags)
	if err != nil {
		return err
	}
	if err := checkTarget(in.dir); err != nil {
		return err
	}
	return initPackage(c.Context(), cfg, in)
}

// collectInputs classifies the positionals, merges --from, prompts for what
// is missing, and checks every input before any registry access.
func collectInputs(c *cobra.Command, args []string, flags initFlags) (*initInputs, error) {
	name, pathArg := classifyInitArgs(args)
	if pathArg != "" && flags.from != "" {
		return nil, cmdutil.ValidationError(
			fmt.Sprintf("the module path was named more than once: %s, --from %s", pathArg, flags.from),
			"Name it once, positionally or with --from.")
	}
	if pathArg == "" {
		pathArg = flags.from
	}

	var err error
	in := &initInputs{namespace: flags.namespace}
	if in.name, err = promptIfMissing(c, name, "Instance name: ", "instance name", missingName); err != nil {
		return nil, err
	}
	if err := checkName("instance name", in.name); err != nil {
		return nil, err
	}
	if pathArg, err = promptIfMissing(c, pathArg, "Module path (e.g. opmodel.dev/modules/web_app): ", "module path", missingModulePath); err != nil {
		return nil, err
	}
	if in.path, err = modref.ParsePath(pathArg); err != nil {
		return nil, initError(err)
	}
	if in.namespace, err = promptIfMissing(c, in.namespace, "Namespace: ", "namespace", missingNamespace); err != nil {
		return nil, err
	}
	if err := checkName("namespace", in.namespace); err != nil {
		return nil, err
	}
	if in.selector, err = modref.ParseSelector(flags.version); err != nil {
		return nil, initError(err)
	}

	in.packageModulePath = flags.modulePath
	if in.packageModulePath == "" {
		in.packageModulePath = instinit.DefaultPackageModulePath(in.name)
	}
	if err := module.CheckPath(in.packageModulePath); err != nil {
		return nil, cmdutil.ValidationError(fmt.Sprintf("--module-path %q is not a module path: %v", in.packageModulePath, err),
			"A module path carries its major:  example.com/deploy/web@v0")
	}

	in.dir = flags.dir
	if in.dir == "" {
		in.dir = in.name
	}
	return in, nil
}

// classifyInitArgs maps the positionals onto (instance name, module path):
// with two, the order is fixed; with one, a '/' or '.' makes it the module
// path and anything else the name.
func classifyInitArgs(args []string) (name, path string) {
	switch len(args) {
	case 1:
		if strings.ContainsAny(args[0], "/.") {
			return "", args[0]
		}
		return args[0], ""
	case 2:
		return args[0], args[1]
	}
	return "", ""
}

// missingName, missingModulePath and missingNamespace are the refusals for an
// input that cannot be prompted for.
var (
	missingName = publish.Refusal{
		Headline: "no instance name given and standard input is not a terminal",
		Action:   "Pass it:  opm instance init <instance-name> <module-path> --namespace <ns>",
	}
	missingModulePath = publish.Refusal{
		Headline: "no module path given and standard input is not a terminal",
		Action:   "Pass it positionally or with --from:  opm instance init <instance-name> <module-path> --namespace <ns>",
	}
	missingNamespace = publish.Refusal{
		Headline: "standard input is not a terminal, so the namespace cannot be asked",
		Action:   "Pass it:  opm instance init <instance-name> <module-path> --namespace <ns>",
	}
)

// promptIfMissing returns value, or asks for it (what names it) on a
// terminal; without one the missing input is refused.
func promptIfMissing(c *cobra.Command, value, prompt, what string, refusal publish.Refusal) (string, error) {
	if value != "" {
		return value, nil
	}
	r, interactive := cmdutil.StdinReader(c)
	if !interactive {
		return "", cmdutil.Refuse(refusal)
	}
	answer, err := cmdutil.PromptLine(r, prompt, what)
	if err != nil {
		return "", err
	}
	if answer == "" {
		return "", cmdutil.ValidationError(what+" must not be empty", "Rerun and answer the prompt, or pass it as an argument.")
	}
	return answer, nil
}

// checkName applies core's #NameType to an instance name or namespace.
func checkName(what, value string) error {
	if len(value) <= 63 && nameRule.MatchString(value) {
		return nil
	}
	return cmdutil.ValidationError(fmt.Sprintf("%s %q is invalid: %s", what, value, nameRuleText),
		"Example:  web-app")
}

// checkTarget refuses a directory that exists, or one that would sit inside
// another CUE module (0016:D5:R5, 0016:D10:R1).
func checkTarget(dir string) error {
	if _, err := os.Lstat(dir); err == nil {
		return cmdutil.Refuse(publish.Refusal{
			Headline: dir + " already exists",
			Action:   "Choose another directory with --dir.",
		})
	} else if !errors.Is(err, os.ErrNotExist) {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("checking %s: %w", dir, err)}
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("resolving %s: %w", dir, err)}
	}
	if root := loader.ModuleRootFrom(filepath.Dir(abs)); root != "" {
		return cmdutil.Refuse(publish.Refusal{
			Headline:    fmt.Sprintf("%s is inside the CUE module at %s", dir, root),
			Consequence: "An instance package is its own CUE module; nested in another module's\ntree, that module's tooling walks into it.",
			Action:      "Initialize outside it, e.g. with --dir pointing elsewhere.",
		})
	}
	return nil
}

// initPackage resolves and acquires the module, renders the package and
// writes it, then reports.
func initPackage(ctx context.Context, cfg *config.GlobalConfig, in *initInputs) error {
	route := modref.Route(cfg.Registry, in.path)
	src, err := modref.NewSource(cfg.Registry)
	if err != nil {
		return initError(err)
	}
	res, err := modref.Resolve(ctx, src, modref.Request{
		Path:      in.path,
		Selector:  in.selector,
		CoreMajor: cmdutil.CoreMajor(),
		Registry:  route,
	})
	if err != nil {
		return initError(err)
	}
	for _, line := range modref.Report(res) {
		output.Info(line)
	}

	modVersion, err := module.NewVersion(res.Import(), res.Version)
	if err != nil {
		return initError(fmt.Errorf("forming module version %s@%s: %w", res.Import(), res.Version, err))
	}
	core, err := corePin(ctx, src, modVersion, route)
	if err != nil {
		return initError(err)
	}

	mod, err := acquireModule(ctx, config.NewKernel(cfg.Registry), modVersion, route)
	if err != nil {
		return initError(err)
	}

	values, source, err := instinit.PickValues(mod.Package)
	if err != nil {
		return initError(err)
	}
	files, err := instinit.Render(instinit.Input{
		Name:              in.name,
		Namespace:         in.namespace,
		PackageModulePath: in.packageModulePath,
		Module:            modVersion,
		Core:              core,
		Values:            values,
		Source:            source,
	})
	if err != nil {
		return initError(err)
	}
	if err := instinit.Write(ctx, in.dir, files, cfg.Registry); err != nil {
		return initError(err)
	}

	report(in.dir, source, instinit.IsEmpty(values))
	return nil
}

// acquireModule acquires the resolved module through the kernel. A registry
// that gave no response is a *publish.ConnectivityError (exit 3); any other
// failure, a registry answer among them, is "loading <module>" (exit 1).
func acquireModule(ctx context.Context, k *kernel.Kernel, mv module.Version, route string) (*libmodule.Module, error) {
	mod, err := k.AcquireModuleFromRegistry(ctx, mv.Path(), mv.Version())
	if err != nil {
		if cuemod.IsConnectivityError(err) {
			return nil, &publish.ConnectivityError{Op: fmt.Sprintf("fetching %s (registry %s)", mv, route), Err: err}
		}
		return nil, fmt.Errorf("loading %s: %w", mv, err)
	}
	return mod, nil
}

// corePin reads the module's own opmodel.dev/core dependency from its
// published module file.
func corePin(ctx context.Context, src modref.Source, mv module.Version, route string) (module.Version, error) {
	mf, err := src.ModFile(ctx, mv)
	if err != nil {
		return module.Version{}, &publish.ConnectivityError{
			Op:  fmt.Sprintf("reading the module file of %s (registry %s)", mv, route),
			Err: err,
		}
	}
	for dep, d := range mf.Deps {
		if p, _, ok := strings.Cut(dep, "@"); ok && p == corePath {
			return module.NewVersion(dep, d.Version)
		}
	}
	return module.Version{}, &modref.RefusalError{Refusal: publish.Refusal{
		Headline: fmt.Sprintf("%s declares no %s dependency", mv, corePath),
		Action:   "Choose a release of an OPM module; every one depends on core.",
	}}
}

// report prints what was written and the next step.
func report(dir string, source instinit.ValuesSource, empty bool) {
	vetPath := filepath.Join(dir, instinit.InstanceFile)
	switch source {
	case instinit.FromInitValues:
		output.Println("Values template: initValues (the module's starting values; edit them for this instance)")
	case instinit.FromDebugValues:
		output.Println("Values template: debugValues (module declares no initValues; review before deploying)")
	case instinit.FromEmpty:
		output.Println("Values template: empty (module offers no concrete starting values)")
	}
	if empty {
		output.Warn(fmt.Sprintf("values.cue is empty; run 'opm instance vet %s' to see what the module requires", vetPath))
	}
	output.Println(fmt.Sprintf("Initialized instance %s/", filepath.ToSlash(dir)))
	for _, f := range []string{instinit.ModuleFile, instinit.InstanceFile, instinit.ValuesFile} {
		output.Println("  " + f)
	}
	output.Println(fmt.Sprintf("\nValidate it:  opm instance vet %s", vetPath))
}

// initError maps errors onto the house funnels: refusals print and exit 2,
// connectivity failures exit 3, anything else exits 1.
func initError(err error) error {
	var exitErr *opmexit.ExitError
	if errors.As(err, &exitErr) {
		return err
	}
	var refusalErr *modref.RefusalError
	if errors.As(err, &refusalErr) {
		return cmdutil.Refuse(refusalErr.Refusal)
	}
	var connErr *publish.ConnectivityError
	if errors.As(err, &connErr) {
		return &opmexit.ExitError{Code: opmexit.ExitConnectivityError, Err: err}
	}
	return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
}

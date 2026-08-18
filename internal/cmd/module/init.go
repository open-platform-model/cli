package modulecmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/publish"
	"github.com/open-platform-model/cli/internal/scaffold"
)

// NewModuleInitCmd creates the module init command: fetch-based scaffolding
// from published template modules, and repair of an existing tree (0011 D20,
// D25; the cli-template-modules change).
func NewModuleInitCmd(cfg *config.GlobalConfig) *cobra.Command {
	var fromFlag, templateFlag, dirFlag string
	var yesFlag bool

	c := &cobra.Command{
		Use:   "init [new-module-path] [template]",
		Short: "Scaffold a module from a published template, or repair an existing tree",
		Long: `Create a new OPM module by fetching a published template module and
re-identifying it to your module path, or repair an existing module tree.

	A bare-word template (letters, digits, underscores) is a shortcut into the
	official set at opmodel.dev/templates/<name>; run 'opm module template list'
	to see it. A reference containing '/' or '.' is a literal module path and is
	never expanded — any published module can seed a scaffold. An '@vN' suffix
	floats to the newest release within that major (stable preferred), a full
	SemVer pins the exact tag, and no suffix takes the template's default major.

	Scaffolding requires the registry once for an uncached template; after any
	successful fetch, CUE's module cache serves repeats offline. No template is
	embedded in the binary.

	Run against a directory that already holds a module, init detects a missing
	or disagreeing cue.mod module line or identity package, shows exactly what
	it would create or edit, and asks before writing (D20). It never invents
	identity: the module path and version always come from the tree or from
	your arguments.

	Exit codes: 0 scaffolded or repaired, 2 refused, 3 registry unreachable.

	Examples:
	  # Scaffold from the default template (standard)
	  opm mod init example.com/modules/my_app@v0

	  # A specific official template, floating within its v1 line
	  opm mod init example.com/modules/my_app@v0 standard@v1

	  # Prompt for the module path (interactive)
	  opm mod init standard

	  # Clone any published module as the starting point
	  opm mod init example.com/modules/my_app@v0 --from example.com/modules/donor@v2

	  # Repair the module in the current directory
	  opm mod init`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			return runModuleInit(c, cfg, args, initFlags{
				from: fromFlag, template: templateFlag, dir: dirFlag, yes: yesFlag,
			})
		},
	}

	c.Flags().StringVar(&fromFlag, "from", "", "Template reference or any published module path to clone")
	c.Flags().StringVarP(&templateFlag, "template", "t", "", "Alias for --from")
	c.Flags().StringVarP(&dirFlag, "dir", "d", "", "Module directory (defaults to the module path's leaf)")
	c.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Repair without the second confirmation")

	return c
}

type initFlags struct {
	from, template, dir string
	yes                 bool
}

// runModuleInit routes one invocation: classify the positionals by shape
// (a module path has '/' or '.'; a template ref may not), then scaffold into
// a fresh directory or repair an existing one.
func runModuleInit(c *cobra.Command, cfg *config.GlobalConfig, args []string, flags initFlags) error {
	pathArg, templateArg, err := classifyArgs(args)
	if err != nil {
		return err
	}

	templateRef, err := pickTemplateRef(templateArg, flags)
	if err != nil {
		return err
	}

	// Interactive form: a template named but no module path — ask for one
	// (D20's "asks for one"); without a terminal there is nothing to ask.
	if pathArg == "" && templateRef != "" {
		pathArg, err = promptModulePath(c)
		if err != nil {
			return err
		}
	}

	targetDir := flags.dir
	if targetDir == "" {
		if pathArg == "" {
			targetDir = "."
		} else {
			targetDir = scaffold.Leaf(pathArg)
		}
	}

	if moduleish(targetDir) {
		if templateRef != "" {
			return validationError(fmt.Sprintf("%s already holds a module; a template only seeds a new tree", targetDir),
				"Rerun without the template to repair the existing tree, or scaffold\nelsewhere with --dir.")
		}
		return runRepair(c, cfg, targetDir, pathArg, flags.yes)
	}

	// No path, no template, and nothing to repair: an empty invocation in an
	// empty directory. Prompt for the path and scaffold the default.
	if pathArg == "" {
		pathArg, err = promptModulePath(c)
		if err != nil {
			return err
		}
		if flags.dir == "" {
			targetDir = scaffold.Leaf(pathArg)
		}
	}
	if templateRef == "" {
		templateRef = scaffold.DefaultTemplate
	}

	return runScaffold(c, cfg, pathArg, templateRef, targetDir)
}

// classifyArgs maps the positionals onto (new module path, template ref) by
// shape: with two, the first is the path; with one, a '/' or '.' makes it
// the path and a bare word makes it the template.
func classifyArgs(args []string) (pathArg, templateArg string, err error) {
	switch len(args) {
	case 0:
		return "", "", nil
	case 1:
		if pathShaped(args[0]) {
			return args[0], "", nil
		}
		return "", args[0], nil
	case 2:
		if !pathShaped(args[0]) {
			return "", "", validationError(fmt.Sprintf("the first argument must be the new module path; %q is not one", args[0]),
				"With two arguments the order is:  opm mod init <new-module-path> <template>")
		}
		return args[0], args[1], nil //nolint:gosec // G602: the enclosing case is len(args) == 2
	default:
		// Unreachable behind cobra.MaximumNArgs(2).
		return "", "", validationError("too many arguments", "Usage:  opm mod init [new-module-path] [template]")
	}
}

// pathShaped mirrors the template-ref grammar's discriminator: a reference
// containing '/' or '.' before any '@' suffix is a module path.
func pathShaped(s string) bool {
	head := s
	if i := strings.LastIndex(s, "@"); i >= 0 {
		head = s[:i]
	}
	return strings.ContainsAny(head, "/.")
}

// pickTemplateRef merges the positional template with the --from / -t
// spellings; naming two is refused rather than ranked.
func pickTemplateRef(templateArg string, flags initFlags) (string, error) {
	var set []string
	for _, s := range []string{templateArg, flags.from, flags.template} {
		if s != "" {
			set = append(set, s)
		}
	}
	switch len(set) {
	case 0:
		return "", nil
	case 1:
		return set[0], nil
	default:
		return "", validationError(fmt.Sprintf("the template is named more than once: %s", strings.Join(set, ", ")),
			"Name it once — positionally, or via --from / -t.")
	}
}

// moduleish reports whether dir exists and holds module content worth
// repairing: a cue.mod directory, an identity package, or any .cue file at
// the root.
func moduleish(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "cue.mod")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "identity", "identity.cue")); err == nil {
		return true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".cue") {
			return true
		}
	}
	return false
}

// runScaffold is the fetch-and-re-identify path.
func runScaffold(c *cobra.Command, cfg *config.GlobalConfig, newPath, templateRef, targetDir string) error {
	ref, err := scaffold.ParseTemplateRef(templateRef)
	if err != nil {
		return validationError(err.Error(), "List the official templates:  opm module template list")
	}
	if err := scaffold.ValidateNewModulePath(newPath); err != nil {
		return validationError(err.Error(), "Example:  opm mod init example.com/modules/my_app@v0")
	}
	if _, err := os.Stat(targetDir); err == nil {
		return validationError(fmt.Sprintf("directory already exists: %s", targetDir),
			"Choose another directory (--dir), or run inside a module tree to repair it.")
	}

	k := kernel.New(kernel.WithRegistry(cfg.Registry))
	result, err := scaffold.Run(c.Context(), k, cfg.Registry, newPath, ref, targetDir)
	if err != nil {
		return initError(err)
	}

	output.Println(fmt.Sprintf("Scaffolded %s from %s %s\n", newPath, result.TemplatePath, result.TemplateVersion))
	entries := make([]output.FileEntry, 0, len(result.Files)+1)
	entries = append(entries, output.FileEntry{Path: result.Dir + "/", Description: "Module directory"})
	for _, f := range result.Files {
		entries = append(entries, output.FileEntry{Path: "  " + f})
	}
	output.Print(output.RenderFileTree(entries, 30))
	output.Println(fmt.Sprintf("\nValidate it:  opm module vet %s", result.Dir))
	return nil
}

// runRepair is the adopt-and-repair path (D20): detect, show every file to
// be created or edited, confirm a second time, apply.
func runRepair(c *cobra.Command, cfg *config.GlobalConfig, dir, pathArg string, yes bool) error {
	k := kernel.New(kernel.WithRegistry(cfg.Registry))
	plan, err := scaffold.DetectRepair(c.Context(), k, cfg.Registry, dir, pathArg)
	if err != nil {
		return initError(err)
	}

	if len(plan.Actions) == 0 {
		output.Println(fmt.Sprintf("Nothing to repair in %s — cue.mod and identity agree on %s.", dir, plan.ModulePath))
		output.Println(fmt.Sprintf("Full conformance check:  opm module vet %s", dir))
		return nil
	}

	output.Print(plan.Describe())
	if !yes {
		ok, err := confirm(c, "Apply these repairs? [y/N]: ")
		if err != nil {
			return err
		}
		if !ok {
			return validationError("repair declined", "Rerun with --yes to skip this confirmation.")
		}
	}

	if err := plan.Apply(); err != nil {
		return err
	}
	output.Println(fmt.Sprintf("Repaired %s to %s.", dir, plan.ModulePath))
	output.Println(fmt.Sprintf("Full conformance check:  opm module vet %s", dir))
	return nil
}

// promptModulePath asks for the new module path — interactively on a
// terminal, from injected input under test, and refusing on a plain
// non-interactive stdin.
func promptModulePath(c *cobra.Command) (string, error) {
	r, interactive := stdinReader(c)
	if !interactive {
		return "", refusal(publish.Refusal{
			Headline:    "no module path given and standard input is not a terminal",
			Consequence: "The template-only form prompts for the new module path; a pipe has nobody\nto ask.",
			Action:      "Pass it:  opm mod init <new-module-path> <template>",
		})
	}
	output.Prompt("New module path (e.g. example.com/modules/my_app@v0): ")
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading module path: %w", err)
	}
	path := strings.TrimSpace(line)
	if path == "" {
		return "", validationError("module path must not be empty", "Example:  example.com/modules/my_app@v0")
	}
	return path, nil
}

// confirm reads a yes/no answer, refusing on a plain non-interactive stdin —
// the second confirmation cannot be defaulted through a pipe (D20); --yes is
// the explicit bypass.
func confirm(c *cobra.Command, prompt string) (bool, error) {
	r, interactive := stdinReader(c)
	if !interactive {
		return false, refusal(publish.Refusal{
			Headline:    "standard input is not a terminal, so the repair confirmation cannot be asked",
			Consequence: "Repair writes into a tree you own; the second confirmation is the whole\nof the safety mechanism (D20).",
			Action:      "Rerun with --yes to consent up front.",
		})
	}
	output.Prompt(prompt)
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

// stdinReader returns the command's input and whether it may be prompted:
// an injected reader (tests) always may; the process stdin only when it is a
// terminal.
func stdinReader(c *cobra.Command) (io.Reader, bool) {
	in := c.InOrStdin()
	if f, ok := in.(*os.File); ok && f == os.Stdin {
		//nolint:gosec // G115: stdin's fd is a small non-negative number on every supported platform
		return in, term.IsTerminal(int(os.Stdin.Fd()))
	}
	return in, true
}

// initError maps scaffold errors onto the house funnels: refusals print and
// exit 2, connectivity failures exit 3, the rest exit 1.
func initError(err error) error {
	var refusalErr *scaffold.RefusalError
	if errors.As(err, &refusalErr) {
		return refusal(refusalErr.Refusal)
	}
	var connErr *publish.ConnectivityError
	if errors.As(err, &connErr) {
		return &opmexit.ExitError{Code: opmexit.ExitConnectivityError, Err: err}
	}
	return err
}

// refusal prints one refusal through the house funnel and exits 2.
func refusal(r publish.Refusal) error {
	cmdutil.PrintRefusals([]publish.Refusal{r})
	return &opmexit.ExitError{
		Code:    opmexit.ExitValidationError,
		Err:     errors.New(r.Headline),
		Printed: true,
	}
}

// validationError is a plain exit-2 error with a hint.
func validationError(msg, hint string) error {
	return &opmexit.ExitError{
		Code: opmexit.ExitValidationError,
		Err:  fmt.Errorf("%s\n  %s", msg, hint),
	}
}

package modulecmd

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"github.com/spf13/cobra"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// NewModuleEvalCmd creates the module eval command.
func NewModuleEvalCmd(cfg *config.GlobalConfig) *cobra.Command {
	var expression string

	c := &cobra.Command{
		Use:   "eval [path]",
		Short: "Print the evaluated module as CUE",
		Long: `Evaluate an OPM module and print the result as formatted CUE.

The module directory is loaded as 'opm module vet' and 'opm module build'
load it, through the kernel's module acquire, so imports resolve from the
module's own cue.mod, its local replacements and the configured registry.
Nothing is validated beyond that load and nothing is rendered: no platform,
no cluster, no values. Definitions and attributes are kept and fields that
are not concrete are printed as written, so this shows the module as
authored after evaluation.

Use -e/--expression to print only the value at a CUE path, such as #config
or metadata.name.

Arguments:
  path    Path to module directory (default: current directory)

Examples:
  # Print the evaluated module in the current directory
  opm module eval

  # Print the configuration schema of a module
  opm module eval ./my-module -e '#config'

  # Print one metadata field
  opm module eval ./my-module -e metadata.name`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runModuleEval(c.Context(), c.OutOrStdout(), cfg, cmdutil.ResolveModulePath(args), expression)
		},
	}

	c.Flags().StringVarP(&expression, "expression", "e", "", "Print only the value at this CUE path, e.g. #config or metadata.name")

	return c
}

// runModuleEval loads the module directory through the kernel, selects the
// value at expression (the whole module when empty) and writes it to out as
// formatted CUE.
func runModuleEval(ctx context.Context, out io.Writer, cfg *config.GlobalConfig, modulePath, expression string) error {
	if err := cmdutil.ValidateModuleInputPath(modulePath); err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}

	if ctx == nil {
		ctx = context.Background()
	}
	k := config.NewKernel(cfg.Registry)
	mod, err := k.AcquireModuleFromDir(ctx, modulePath)
	if err != nil {
		return &opmexit.ExitError{
			Code: opmexit.ExitGeneralError,
			Err:  fmt.Errorf("loading module from %s: %w", modulePath, err),
		}
	}

	v, err := selectExpression(mod.Package, expression)
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}

	data, err := format.Node(evalSyntax(v))
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("formatting evaluated module: %w", err)}
	}
	// format.Node leaves an expression (a selected scalar) without a trailing
	// newline; a file already ends in one.
	if !bytes.HasSuffix(data, []byte("\n")) {
		data = append(data, '\n')
	}
	if _, err := out.Write(data); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}

// evalSyntax renders v as CUE syntax: definitions and attributes kept,
// defaults resolved, fields that are not concrete left as written. A struct
// prints as its bare fields, as `cue eval` prints it, so the whole module
// reads as a CUE file rather than one braced expression.
func evalSyntax(v cue.Value) ast.Node {
	node := v.Syntax(cue.Final(), cue.Concrete(false), cue.Definitions(true), cue.Attributes(true))
	if lit, ok := node.(*ast.StructLit); ok {
		return &ast.File{Decls: lit.Elts}
	}
	return node
}

// selectExpression returns the value of root at the CUE path expression, or
// root itself when expression is empty. A path that does not parse or that
// names nothing in the module is an error saying so.
func selectExpression(root cue.Value, expression string) (cue.Value, error) {
	if expression == "" {
		return root, nil
	}
	p := cue.ParsePath(expression)
	if err := p.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("invalid expression %q: %w", expression, err)
	}
	v := root.LookupPath(p)
	if !v.Exists() {
		return cue.Value{}, fmt.Errorf("expression %q does not exist in the module", expression)
	}
	return v, nil
}

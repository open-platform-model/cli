// Command cmdref writes the opm command reference, docs/site/reference/cli/,
// from the CLI's cobra command tree, or with -check reports the pages that
// are stale and exits 1.
//
// Run it from the repository root: task docs:reference regenerates,
// task docs:reference:check checks.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/open-platform-model/cli/internal/cmd"
	"github.com/open-platform-model/cli/internal/cmdref"
)

func main() {
	check := flag.Bool("check", false, "write nothing; exit 1 when a page is stale")
	dir := flag.String("dir", "docs/site/reference/cli", "reference directory")
	flag.Parse()

	if err := run(*dir, *check); err != nil {
		fmt.Fprintln(os.Stderr, "cmdref:", err)
		os.Exit(1)
	}
}

func run(dir string, check bool) error {
	root := cmd.NewRootCmd()
	// cobra adds the completion command when the CLI executes; add it here so
	// the reference lists what `opm --help` lists.
	root.InitDefaultCompletionCmd()

	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	pages, err := cmdref.Generate(root, cmdref.Options{Home: home})
	if err != nil {
		return err
	}
	stale, err := cmdref.Sync(dir, pages, check)
	if err != nil {
		return err
	}
	if !check {
		for _, path := range stale {
			fmt.Println("updated", path)
		}
		return nil
	}
	if len(stale) > 0 {
		for _, path := range stale {
			fmt.Fprintln(os.Stderr, "stale:", path)
		}
		return fmt.Errorf("the command reference is stale; run task docs:reference and commit the result")
	}
	fmt.Println("command reference is current:", dir)
	return nil
}

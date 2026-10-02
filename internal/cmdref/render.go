// Package cmdref generates the opm command reference, the site pages under
// docs/site/reference/cli/, from the cobra command tree.
//
// The pages follow the site page dialect (workspace STYLE.md, "Site Pages"):
// four front-matter keys, root-absolute links with a trailing slash, a
// language tag on every code fence, no raw HTML. Each page's generated part
// sits between marker comments, so text an author adds around it survives a
// regeneration. Every generated fact comes from the command tree: a command's
// Use, Short, Long, Example and Aliases fields and its flags.
package cmdref

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// SitePath is the address of the section the pages publish under.
const SitePath = "/docs/reference/cli/"

// Page is one generated file of the reference.
type Page struct {
	// Name is the file name inside the reference directory.
	Name string
	// FrontMatter is the page's front matter, delimiters included.
	FrontMatter string
	// Body is the generated block, without its marker comments.
	Body string
}

// Options tunes the output so it does not depend on the machine it runs on.
type Options struct {
	// Home is the user's home directory; a flag default under it is written
	// with a leading ~ instead.
	Home string
}

var reFileName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*\.md$`)

// Generate renders the reference for root: the section page, _index.md, with
// the root's persistent flags, and one reference page per top-level command
// holding an entry for that command and for every command under it. Hidden,
// deprecated and help-topic commands, and cobra's help command, are left out,
// as `opm --help` leaves them out.
func Generate(root *cobra.Command, opts Options) ([]Page, error) {
	// cobra merges a parent's persistent flags into a command only when it
	// runs; merge them everywhere first, so every use line counts them as
	// `--help` does.
	walk(root, func(c *cobra.Command) { c.InheritedFlags() })
	pages := []Page{indexPage(root, opts)}
	for _, top := range visibleChildren(root) {
		name := strings.ReplaceAll(top.CommandPath(), " ", "-") + ".md"
		if !reFileName.MatchString(name) {
			return nil, fmt.Errorf("command %q: page name %q is not kebab-case", top.CommandPath(), name)
		}
		pages = append(pages, Page{
			Name:        name,
			FrontMatter: frontMatter(top.CommandPath(), sentence(top.Short), "reference", 0),
			Body:        escapeShortcodes(commandPage(root, top, name, opts)),
		})
	}
	return pages, nil
}

func indexPage(root *cobra.Command, opts Options) Page {
	var b strings.Builder
	if long := parseLong(root.Long); len(long.desc) > 0 {
		writeBlocks(&b, long.desc)
		b.WriteString("\n")
	}
	b.WriteString("Each page in this section covers one top-level command and every command under it: " +
		"its usage, description, flags and examples, generated from the CLI's cobra commands. " +
		"`" + root.Name() + " <command> --help` prints the same facts for the CLI you have installed.\n\n")
	b.WriteString(fence("text", usageLines(root)))
	if flags := globalFlags(root); len(flags) > 0 {
		b.WriteString("\n## Global flags\n\nEvery command takes these flags.\n\n")
		writeFlagTable(&b, flags, opts)
	}
	return Page{
		Name:        "_index.md",
		FrontMatter: frontMatter("CLI Reference", "Every opm command and flag, generated from the CLI's cobra commands.", "", 2),
		Body:        escapeShortcodes(b.String()),
	}
}

func commandPage(root, top *cobra.Command, page string, opts Options) string {
	var b strings.Builder
	if len(globalFlags(root)) > 0 {
		b.WriteString("Every command on this page also takes the [global flags](" + SitePath + "#global-flags).\n")
	}
	walk(top, func(c *cobra.Command) {
		b.WriteString("\n")
		writeEntry(&b, root, c, page, opts)
	})
	return b.String()
}

// writeEntry writes one command's entry. Its parts come in a fixed order, each
// only when the command has it: summary, usage (with aliases), description,
// flags, examples, subcommands.
func writeEntry(b *strings.Builder, root, c *cobra.Command, page string, opts Options) {
	b.WriteString("## " + c.CommandPath() + "\n\n")
	if c.Short != "" {
		b.WriteString(sentence(formatProse(c.Short)) + "\n\n")
	}
	b.WriteString(fence("text", usageLines(c)))
	if len(c.Aliases) > 0 {
		spans := make([]string, len(c.Aliases))
		for i, a := range c.Aliases {
			spans[i] = codeSpan(a)
		}
		b.WriteString("\nAliases: " + strings.Join(spans, ", ") + ".\n")
	}

	long := parseLong(c.Long)
	if len(long.desc) > 0 {
		b.WriteString("\n")
		writeBlocks(b, long.desc)
	}
	if flags := commandFlags(root, c); len(flags) > 0 {
		b.WriteString("\n**Flags**\n\n")
		writeFlagTable(b, flags, opts)
	}
	examples := long.examples
	if c.Example != "" {
		if len(examples) > 0 {
			examples = append(examples, "")
		}
		examples = append(examples, dedent(splitLines(c.Example))...)
	}
	if len(examples) > 0 {
		b.WriteString("\n**Examples**\n\n" + fence("sh", examples))
	}
	if subs := visibleChildren(c); len(subs) > 0 {
		b.WriteString("\n**Subcommands**\n\n| Command | Summary |\n| --- | --- |\n")
		for _, s := range subs {
			link := "[" + s.CommandPath() + "](" + SitePath + strings.TrimSuffix(page, ".md") + "/#" + anchor(s.CommandPath()) + ")"
			b.WriteString("| " + link + " | " + cell(sentence(formatProse(s.Short))) + " |\n")
		}
	}
}

func writeBlocks(b *strings.Builder, blocks []block) {
	for i, bl := range blocks {
		if i > 0 {
			b.WriteString("\n")
		}
		if bl.pre {
			b.WriteString(fence(fenceLang(bl.lines), bl.lines))
			continue
		}
		b.WriteString(formatProse(bl.lines[0]) + "\n")
	}
}

// usageLines mirrors cobra's usage template: the use line of a runnable
// command, then "<path> [command]" for a command with subcommands.
func usageLines(c *cobra.Command) []string {
	var lines []string
	if c.Runnable() {
		lines = append(lines, c.UseLine())
	}
	if c.HasAvailableSubCommands() {
		lines = append(lines, c.CommandPath()+" [command]")
	}
	if len(lines) == 0 {
		lines = append(lines, c.CommandPath())
	}
	return lines
}

// visibleChildren returns the subcommands `--help` lists, sorted by name.
func visibleChildren(c *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	for _, s := range c.Commands() {
		if s.IsAvailableCommand() && s.Name() != "help" {
			out = append(out, s)
		}
	}
	return out
}

// walk visits c and then every visible command under it, depth first, in
// name order.
func walk(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for _, s := range visibleChildren(c) {
		walk(s, fn)
	}
}

// globalFlags are the root's persistent flags, which every command takes.
func globalFlags(root *cobra.Command) []*pflag.Flag {
	return visibleFlags(root.PersistentFlags(), nil)
}

// commandFlags are the flags of c that are not global: its own, and any it
// inherits from a parent other than the root.
func commandFlags(root, c *cobra.Command) []*pflag.Flag {
	if c == root {
		return visibleFlags(c.LocalNonPersistentFlags(), nil)
	}
	isGlobal := func(f *pflag.Flag) bool { return root.PersistentFlags().Lookup(f.Name) != nil }
	flags := visibleFlags(c.LocalFlags(), isGlobal)
	flags = append(flags, visibleFlags(c.InheritedFlags(), isGlobal)...)
	sortFlags(flags)
	return flags
}

func visibleFlags(fs *pflag.FlagSet, skip func(*pflag.Flag) bool) []*pflag.Flag {
	var out []*pflag.Flag
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Deprecated != "" || f.Name == "help" || (skip != nil && skip(f)) {
			return
		}
		out = append(out, f)
	})
	sortFlags(out)
	return out
}

func sortFlags(flags []*pflag.Flag) {
	sort.Slice(flags, func(i, j int) bool { return flags[i].Name < flags[j].Name })
}

func writeFlagTable(b *strings.Builder, flags []*pflag.Flag, opts Options) {
	b.WriteString("| Flag | Shorthand | Type | Default | Description |\n| --- | --- | --- | --- | --- |\n")
	for _, f := range flags {
		short := ""
		if f.Shorthand != "" && f.ShorthandDeprecated == "" {
			short = codeSpan("-" + f.Shorthand)
		}
		def := ""
		if d := defaultValue(f, opts); d != "" {
			def = codeSpan(d)
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n",
			cell(codeSpan("--"+f.Name)), cell(short), cell(f.Value.Type()), cell(def), cell(sentence(formatProse(f.Usage))))
	}
}

// defaultValue is the default `--help` would print, or "" where it prints
// none (a zero value), with the home directory written as ~.
func defaultValue(f *pflag.Flag, opts Options) string {
	d := f.DefValue
	switch d {
	case "", "false", "[]", "0", "0s", "<nil>":
		return ""
	}
	if opts.Home != "" && strings.Contains(d, opts.Home) {
		d = strings.ReplaceAll(d, opts.Home, "~")
	}
	return d
}

// cell keeps a value inside its table column.
func cell(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "|", `\|`), `\\|`, `\|`)
}

// anchor is the heading id Hugo gives an entry heading such as
// "opm module apply".
func anchor(heading string) string {
	return strings.ReplaceAll(strings.ToLower(heading), " ", "-")
}

// sentence ends a summary with a full stop unless it already ends a sentence.
func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasSuffix(s, ".") || strings.HasSuffix(s, "?") || strings.HasSuffix(s, "!") {
		return s
	}
	return s + "."
}

func frontMatter(title, description, typ string, weight int) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: " + yamlQuote(title) + "\n")
	b.WriteString("description: " + yamlQuote(description) + "\n")
	if typ != "" {
		b.WriteString("type: " + typ + "\n")
	}
	if weight > 0 {
		fmt.Fprintf(&b, "weight: %d\n", weight)
	}
	b.WriteString("---\n")
	return b.String()
}

func yamlQuote(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

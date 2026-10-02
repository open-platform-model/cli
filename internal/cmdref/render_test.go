package cmdref

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cmd"
)

func noop(*cobra.Command, []string) error { return nil }

// testTree is a small command tree with one of everything the generator
// handles.
func testTree() *cobra.Command {
	root := &cobra.Command{Use: "opm", Short: "Open Platform Model CLI", Long: "OPM CLI manages modules."}
	root.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	mod := &cobra.Command{Use: "module", Aliases: []string{"mod"}, Short: "Work with module source"}
	mod.PersistentFlags().String("kubeconfig", "", "Path to kubeconfig file")
	build := &cobra.Command{
		Use:   "build [path]",
		Short: "Render a module",
		Long:  "Render a module.\n\nExamples:\n  # Build it\n  opm module build ./m",
		RunE:  noop,
	}
	build.Flags().StringP("output", "o", "yaml", "Output format: yaml, json")
	build.Flags().String("cache", "/home/someone/.opm/cache", "Cache directory")
	build.Flags().Bool("legacy", false, "Old flag")
	_ = build.Flags().MarkDeprecated("legacy", "use --output")
	build.Flags().Bool("secret", false, "Hidden flag")
	_ = build.Flags().MarkHidden("secret")
	mod.AddCommand(build,
		&cobra.Command{Use: "vet", Short: "Validate a module", RunE: noop, Example: "  opm module vet"},
		&cobra.Command{Use: "internal", Short: "Hidden", Hidden: true, RunE: noop},
		&cobra.Command{Use: "old", Short: "Deprecated", Deprecated: "use vet", RunE: noop},
	)
	root.AddCommand(mod, &cobra.Command{Use: "version", Short: "Show version information", RunE: noop})
	return root
}

func pageByName(t *testing.T, pages []Page, name string) Page {
	t.Helper()
	for _, p := range pages {
		if p.Name == name {
			return p
		}
	}
	require.Failf(t, "page not generated", "%s", name)
	return Page{}
}

func TestGenerate_PagesPerTopLevelCommand(t *testing.T) {
	pages, err := Generate(testTree(), Options{Home: "/home/someone"})
	require.NoError(t, err)

	names := make([]string, 0, len(pages))
	for _, p := range pages {
		names = append(names, p.Name)
	}
	assert.Equal(t, []string{"_index.md", "opm-module.md", "opm-version.md"}, names)

	index := pageByName(t, pages, "_index.md")
	assert.Equal(t, "---\ntitle: \"CLI Reference\"\ndescription: \"Every opm command and flag, generated from the CLI's cobra commands.\"\nweight: 2\n---\n", index.FrontMatter)
	assert.Contains(t, index.Body, "## Global flags")
	assert.Contains(t, index.Body, "| `--verbose` | `-v` | bool |  | Enable verbose output. |")

	mod := pageByName(t, pages, "opm-module.md")
	assert.Equal(t, "---\ntitle: \"opm module\"\ndescription: \"Work with module source.\"\ntype: reference\n---\n", mod.FrontMatter)
}

func TestGenerate_EntryParts(t *testing.T) {
	pages, err := Generate(testTree(), Options{Home: "/home/someone"})
	require.NoError(t, err)
	body := pageByName(t, pages, "opm-module.md").Body

	// Entries in tree order; hidden and deprecated commands are left out.
	headings := regexp.MustCompile(`(?m)^## .*$`).FindAllString(body, -1)
	assert.Equal(t, []string{"## opm module", "## opm module build", "## opm module vet"}, headings)

	assert.Contains(t, body, "Aliases: `mod`.")
	assert.Contains(t, body, "```text\nopm module build [path] [flags]\n```")
	assert.Contains(t, body, "**Examples**\n\n```sh\n# Build it\nopm module build ./m\n```")
	assert.Contains(t, body, "**Examples**\n\n```sh\nopm module vet\n```", "cobra's Example field is an example too")
	assert.Contains(t, body, "| [opm module build](/docs/reference/cli/opm-module/#opm-module-build) | Render a module. |")

	// A parent's persistent flag is listed on its children; the root's are not.
	assert.Contains(t, body, "| `--kubeconfig` |  | string |  | Path to kubeconfig file. |")
	assert.NotContains(t, body, "`--verbose`")
	// Defaults: zero values print none, the home directory reads as ~.
	assert.Contains(t, body, "| `--output` | `-o` | string | `yaml` | Output format: yaml, json. |")
	assert.Contains(t, body, "| `--cache` |  | string | `~/.opm/cache` | Cache directory. |")
	assert.NotContains(t, body, "legacy")
	assert.NotContains(t, body, "secret")
	assert.NotContains(t, body, "/home/someone")
}

func TestGenerate_IsDeterministic(t *testing.T) {
	a, err := Generate(testTree(), Options{})
	require.NoError(t, err)
	b, err := Generate(testTree(), Options{})
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

var (
	reFenceOpen   = regexp.MustCompile("^(```+)(.*)$")
	reLink        = regexp.MustCompile(`\]\(([^)]*)\)`)
	reSiteLink    = regexp.MustCompile(`^/docs/([a-z0-9-]+/)*(#[a-z0-9-]+)?$`)
	reFrontMatter = regexp.MustCompile(`^---\n(title: "[^"\n]+"\ndescription: "[^"\n]+"\n)(type: reference\n)?(weight: [1-9][0-9]*\n)?---\n$`)
)

// TestGenerate_RealTreeFollowsThePageDialect checks the dialect rules the
// site build enforces against the pages of the real opm command tree.
func TestGenerate_RealTreeFollowsThePageDialect(t *testing.T) {
	root := cmd.NewRootCmd()
	root.InitDefaultCompletionCmd()
	pages, err := Generate(root, Options{Home: "/home/someone"})
	require.NoError(t, err)
	require.Greater(t, len(pages), 5)

	for _, p := range pages {
		t.Run(p.Name, func(t *testing.T) {
			assert.Regexp(t, reFileName, strings.TrimPrefix(p.Name, "_"))
			assert.Regexp(t, reFrontMatter, p.FrontMatter)
			assert.Equal(t, p.Name != "_index.md", strings.Contains(p.FrontMatter, "type: reference"))
			assert.NotContains(t, p.Body, "{{<")
			assert.NotContains(t, p.Body, "](./")

			open := ""
			for i, line := range strings.Split(p.Body, "\n") {
				m := reFenceOpen.FindStringSubmatch(line)
				switch {
				case open == "" && m != nil:
					assert.NotEmpty(t, m[2], "line %d: code fence without a language tag", i+1)
					open = m[1]
				case open != "" && line == open:
					open = ""
				case open == "":
					assert.NotRegexp(t, `^#[^#\s]|^# `, line, "line %d: a stray H1", i+1)
					for _, l := range reLink.FindAllStringSubmatch(line, -1) {
						assert.Regexp(t, reSiteLink, l[1], "line %d: link", i+1)
					}
				}
			}
			assert.Empty(t, open, "unclosed code fence")
		})
	}
}

func TestSync_WritesChecksAndKeepsAuthoredText(t *testing.T) {
	dir := t.TempDir()
	pages := []Page{{Name: "opm-x.md", FrontMatter: "---\ntitle: \"opm x\"\ndescription: \"X.\"\ntype: reference\n---\n", Body: "## opm x\n"}}

	stale, err := Sync(dir, pages, true)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(dir, "opm-x.md")}, stale, "a missing page is stale")
	assert.NoFileExists(t, filepath.Join(dir, "opm-x.md"), "check writes nothing")

	_, err = Sync(dir, pages, false)
	require.NoError(t, err)
	got, err := os.ReadFile(filepath.Join(dir, "opm-x.md"))
	require.NoError(t, err)
	assert.Equal(t, "---\ntitle: \"opm x\"\ndescription: \"X.\"\ntype: reference\n---\n\n"+BeginMarker+"\n\n## opm x\n\n"+EndMarker+"\n", string(got))

	// Authored text around the block survives a regeneration.
	authored := strings.Replace(string(got), BeginMarker, "An authored lead-in.\n\n"+BeginMarker, 1) + "\n## See also\n\nAuthored.\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "opm-x.md"), []byte(authored), 0o600))
	stale, err = Sync(dir, pages, true)
	require.NoError(t, err)
	assert.Empty(t, stale)

	pages[0].Body = "## opm x\n\nChanged.\n"
	stale, err = Sync(dir, pages, true)
	require.NoError(t, err)
	assert.Len(t, stale, 1)
	_, err = Sync(dir, pages, false)
	require.NoError(t, err)
	got, err = os.ReadFile(filepath.Join(dir, "opm-x.md"))
	require.NoError(t, err)
	assert.Contains(t, string(got), "An authored lead-in.\n\n"+BeginMarker+"\n\n## opm x\n\nChanged.\n\n"+EndMarker+"\n\n## See also\n\nAuthored.\n")
}

func TestSync_OrphansAndAuthoredPages(t *testing.T) {
	dir := t.TempDir()
	orphan := filepath.Join(dir, "opm-gone.md")
	authored := filepath.Join(dir, "guide.md")
	require.NoError(t, os.WriteFile(orphan, []byte("---\ntitle: \"x\"\n---\n\n"+BeginMarker+"\n\n"+EndMarker+"\n"), 0o600))
	require.NoError(t, os.WriteFile(authored, []byte("---\ntitle: \"guide\"\n---\n\nAuthored.\n"), 0o600))

	stale, err := Sync(dir, nil, true)
	require.NoError(t, err)
	assert.Equal(t, []string{orphan}, stale)

	_, err = Sync(dir, nil, false)
	require.NoError(t, err)
	assert.NoFileExists(t, orphan)
	assert.FileExists(t, authored)
}

func TestCompose_RefusesAPageWithoutAGeneratedBlock(t *testing.T) {
	_, err := Compose([]byte("---\ntitle: \"x\"\n---\n\nAuthored only.\n"), Page{Name: "x.md"})
	assert.ErrorContains(t, err, "begin marker missing")
}

func TestListItems(t *testing.T) {
	marker, items, ok := listItems([]string{"- one", "- two"})
	assert.True(t, ok)
	assert.Equal(t, "-", marker)
	assert.Equal(t, []string{"one", "two"}, items)

	marker, items, ok = listItems([]string{"1. first", "2. second"})
	assert.True(t, ok)
	assert.Equal(t, "1.", marker)
	assert.Equal(t, []string{"first", "second"}, items)

	_, _, ok = listItems([]string{"- one", "  continued"})
	assert.False(t, ok, "a continuation line keeps the block preformatted")
	_, _, ok = listItems([]string{"- one", "1. two"})
	assert.False(t, ok, "mixed markers keep the block preformatted")
	_, _, ok = listItems([]string{"0   the module was written"})
	assert.False(t, ok)
}

func TestWriteBlocks_ListBlockBecomesAMarkdownList(t *testing.T) {
	var b strings.Builder
	writeBlocks(&b, parseLong("Show version.\n\nDisplays:\n  - the CLI version\n  - the CUE SDK version").desc)
	assert.Equal(t, "Show version.\n\nDisplays:\n\n- the CLI version\n- the CUE SDK version\n", b.String())
}

package modulecmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/scaffold"
)

// runInit executes the init command with args and optional injected stdin,
// inside dir, returning the error.
func runInit(t *testing.T, dir, stdin string, args ...string) error {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(wd) })

	c := NewModuleInitCmd(&config.GlobalConfig{})
	c.SetArgs(args)
	c.SetOut(new(bytes.Buffer))
	c.SetErr(new(bytes.Buffer))
	if stdin != "" {
		c.SetIn(strings.NewReader(stdin))
	}
	return c.Execute()
}

func exitCode(t *testing.T, err error) int {
	t.Helper()
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	return exitErr.Code
}

func TestClassifyArgs(t *testing.T) {
	p, tpl, err := classifyArgs(nil)
	require.NoError(t, err)
	assert.Empty(t, p)
	assert.Empty(t, tpl)

	p, tpl, err = classifyArgs([]string{"example.com/modules/app@v0"})
	require.NoError(t, err)
	assert.Equal(t, "example.com/modules/app@v0", p)
	assert.Empty(t, tpl)

	p, tpl, err = classifyArgs([]string{"standard@v1"})
	require.NoError(t, err)
	assert.Empty(t, p)
	assert.Equal(t, "standard@v1", tpl)

	p, tpl, err = classifyArgs([]string{"example.com/modules/app@v0", "standard@v1"})
	require.NoError(t, err)
	assert.Equal(t, "example.com/modules/app@v0", p)
	assert.Equal(t, "standard@v1", tpl)

	// Two positionals with a bare word first is a grammar error, not a guess.
	_, _, err = classifyArgs([]string{"standard", "example.com/modules/app@v0"})
	assert.Error(t, err)
}

func TestPickTemplateRef(t *testing.T) {
	ref, err := pickTemplateRef("", initFlags{})
	require.NoError(t, err)
	assert.Empty(t, ref)

	ref, err = pickTemplateRef("standard", initFlags{})
	require.NoError(t, err)
	assert.Equal(t, "standard", ref)

	// -t is an alias for --from; either spelling alone works.
	ref, err = pickTemplateRef("", initFlags{template: "minimal"})
	require.NoError(t, err)
	assert.Equal(t, "minimal", ref)

	ref, err = pickTemplateRef("", initFlags{from: "example.com/modules/donor@v2"})
	require.NoError(t, err)
	assert.Equal(t, "example.com/modules/donor@v2", ref)

	// Naming it twice is refused rather than ranked.
	_, err = pickTemplateRef("standard", initFlags{from: "minimal"})
	assert.Error(t, err)
}

func TestInit_TemplateOnlyNonTTYRefuses(t *testing.T) {
	// No injected stdin: the command sees the process stdin, which is not a
	// terminal under go test — the template-only form must refuse rather
	// than hang or invent a path.
	err := runInit(t, t.TempDir(), "", "standard")
	assert.Equal(t, 2, exitCode(t, err))
	assert.Contains(t, err.Error(), "not a terminal")
}

func TestInit_InvalidNewPathRefusesBeforeAnyIO(t *testing.T) {
	err := runInit(t, t.TempDir(), "", "example.com/modules/My-App@v0")
	assert.Equal(t, 2, exitCode(t, err))
}

func TestInit_TemplateAgainstExistingModuleRefuses(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "app", "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app", "cue.mod", "module.cue"),
		[]byte("module: \"example.com/modules/app@v0\"\n"), 0o644))

	err := runInit(t, dir, "", "example.com/modules/app@v0", "standard")
	assert.Equal(t, 2, exitCode(t, err))
	assert.Contains(t, err.Error(), "already holds a module")
}

func TestInit_RepairConfirmationIsStdinDriven(t *testing.T) {
	// A tree whose identity states the path but whose cue.mod is missing:
	// repair plans one create. "n" declines it; the file must not appear.
	files := map[string]string{
		"identity/identity.cue": `package identity

ModulePath: "example.com/modules/app@v1"
Version:    "1.0.0"
`,
	}
	writeRepairTree := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		mod := filepath.Join(dir, "app")
		for rel, content := range files {
			path := filepath.Join(mod, rel)
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
		}
		return dir
	}

	t.Run("declined leaves the tree untouched", func(t *testing.T) {
		dir := writeRepairTree(t)
		err := runInit(t, dir, "n\n", "--dir", "app")
		assert.Equal(t, 2, exitCode(t, err))
		assert.NoFileExists(t, filepath.Join(dir, "app", "cue.mod", "module.cue"))
	})

	t.Run("accepted applies the plan", func(t *testing.T) {
		dir := writeRepairTree(t)
		err := runInit(t, dir, "y\n", "--dir", "app")
		require.NoError(t, err)
		data, err := os.ReadFile(filepath.Join(dir, "app", "cue.mod", "module.cue"))
		require.NoError(t, err)
		assert.Contains(t, string(data), `module: "example.com/modules/app@v1"`)
	})

	t.Run("--yes bypasses the confirmation", func(t *testing.T) {
		dir := writeRepairTree(t)
		err := runInit(t, dir, "", "--dir", "app", "--yes")
		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(dir, "app", "cue.mod", "module.cue"))
	})

	t.Run("non-TTY without --yes refuses", func(t *testing.T) {
		dir := writeRepairTree(t)
		err := runInit(t, dir, "", "--dir", "app")
		assert.Equal(t, 2, exitCode(t, err))
		assert.NoFileExists(t, filepath.Join(dir, "app", "cue.mod", "module.cue"))
	})
}

// TestInit_TemplateOnlyPromptsForThePath pins the interactive form: the
// template-only invocation reads the new module path from stdin before
// writing anything (D20's "asks for one"). The injected paths steer the run
// into deterministic offline refusals, proving the prompt was consumed and
// nothing was scaffolded.
func TestInit_TemplateOnlyPromptsForThePath(t *testing.T) {
	t.Run("prompted path is used before any write", func(t *testing.T) {
		// The prompted path's leaf directory already holds a module, so the
		// run refuses (template against an existing tree) without touching
		// the registry — reachable only by reading the path from stdin.
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "my_app", "cue.mod"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "my_app", "cue.mod", "module.cue"),
			[]byte("module: \"example.com/modules/my_app@v0\"\n"), 0o644))

		err := runInit(t, dir, "example.com/modules/my_app@v0\n", "standard")
		assert.Equal(t, 2, exitCode(t, err))
		assert.Contains(t, err.Error(), "already holds a module")
	})

	t.Run("empty answer refuses", func(t *testing.T) {
		dir := t.TempDir()
		err := runInit(t, dir, "\n", "standard")
		assert.Equal(t, 2, exitCode(t, err))
		assert.Contains(t, err.Error(), "must not be empty")
		assert.NoDirExists(t, filepath.Join(dir, "standard"))
	})

	t.Run("invalid answer refuses before any write", func(t *testing.T) {
		dir := t.TempDir()
		err := runInit(t, dir, "not a path\n", "standard")
		assert.Equal(t, 2, exitCode(t, err))
		entries, readErr := os.ReadDir(dir)
		require.NoError(t, readErr)
		assert.Empty(t, entries, "nothing may be written on a refused prompt")
	})
}

func TestInit_NothingToRepairIsSuccess(t *testing.T) {
	dir := t.TempDir()
	mod := filepath.Join(dir, "app")
	require.NoError(t, os.MkdirAll(filepath.Join(mod, "identity"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(mod, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mod, "cue.mod", "module.cue"),
		[]byte("module: \"example.com/modules/app@v1\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mod, "identity", "identity.cue"),
		[]byte("package identity\n\nModulePath: \"example.com/modules/app@v1\"\nVersion: \"1.0.0\"\n"), 0o644))

	err := runInit(t, dir, "", "--dir", "app")
	assert.NoError(t, err)
}

// TestTemplateList_SharesTheExpansionTable asserts `template list` prints
// exactly the baked table shortcut expansion reads — one source, no drift.
func TestTemplateList_SharesTheExpansionTable(t *testing.T) {
	c := NewModuleTemplateCmd()
	out := new(bytes.Buffer)
	c.SetArgs([]string{"list"})
	c.SetOut(out)
	require.NoError(t, c.Execute())

	for _, tpl := range scaffold.Official {
		assert.Contains(t, out.String(), tpl.Name)
		assert.Contains(t, out.String(), tpl.DefaultMajor)

		ref, err := scaffold.ParseTemplateRef(tpl.Name)
		require.NoError(t, err)
		assert.Equal(t, scaffold.Segment+"/"+tpl.Name, ref.Base)
	}
}

package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repairTree builds a module tree from relative-path → content.
func repairTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return dir
}

const repairIdentity = `package identity

#VersionType: string & =~"^\\d+"

ModulePath: "example.com/modules/app@v1"
Version:    #VersionType | *"1.2.0"
`

const repairCueMod = `module: "example.com/modules/app@v1"
language: {
	version: "v0.17.0"
}
source: {
	kind: "self"
}
`

func TestDetectRepair(t *testing.T) {
	ctx := context.Background()

	t.Run("aligned tree has nothing to repair", func(t *testing.T) {
		dir := repairTree(t, map[string]string{
			"cue.mod/module.cue":    repairCueMod,
			"identity/identity.cue": repairIdentity,
		})
		plan, err := DetectRepair(ctx, nil, "", dir, "")
		require.NoError(t, err)
		assert.Equal(t, "example.com/modules/app@v1", plan.ModulePath)
		assert.Empty(t, plan.Actions)
	})

	t.Run("missing cue.mod is created from the identity's path", func(t *testing.T) {
		dir := repairTree(t, map[string]string{
			"identity/identity.cue": repairIdentity,
		})
		plan, err := DetectRepair(ctx, nil, "", dir, "")
		require.NoError(t, err)
		require.Len(t, plan.Actions, 1)
		a := plan.Actions[0]
		assert.True(t, a.Create)
		assert.Equal(t, filepath.Join("cue.mod", "module.cue"), a.File)
		assert.Equal(t, "example.com/modules/app@v1", a.New)

		require.NoError(t, plan.Apply())
		data, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
		require.NoError(t, err)
		assert.Contains(t, string(data), `module: "example.com/modules/app@v1"`)
		assert.Contains(t, string(data), `kind: "self"`)
	})

	t.Run("disagreeing identity is realigned to the argument", func(t *testing.T) {
		dir := repairTree(t, map[string]string{
			"cue.mod/module.cue":    repairCueMod,
			"identity/identity.cue": repairIdentity,
		})
		// The argument wins over both; cue.mod also realigns. No self-imports
		// exist, so no stranding.
		plan, err := DetectRepair(ctx, nil, "", dir, "example.com/modules/renamed@v1")
		require.NoError(t, err)
		require.Len(t, plan.Actions, 2)

		require.NoError(t, plan.Apply())
		got, err := DetectRepair(ctx, nil, "", dir, "")
		require.NoError(t, err)
		assert.Equal(t, "example.com/modules/renamed@v1", got.ModulePath)
		assert.Empty(t, got.Actions)
	})

	t.Run("identity and cue.mod disagree with no argument to arbitrate", func(t *testing.T) {
		dir := repairTree(t, map[string]string{
			"cue.mod/module.cue":    `module: "example.com/modules/other@v1"` + "\n",
			"identity/identity.cue": repairIdentity,
		})
		_, err := DetectRepair(ctx, nil, "", dir, "")
		var refusalErr *RefusalError
		require.ErrorAs(t, err, &refusalErr)
		assert.Contains(t, refusalErr.Refusal.Headline, "will not choose")
	})

	t.Run("no path stated anywhere refuses — never invented", func(t *testing.T) {
		dir := repairTree(t, map[string]string{
			"module.cue": "package app\n\nx: 1\n",
		})
		_, err := DetectRepair(ctx, nil, "", dir, "")
		var refusalErr *RefusalError
		require.ErrorAs(t, err, &refusalErr)
		assert.Contains(t, refusalErr.Refusal.Headline, "states no module path")
	})

	t.Run("cue.mod realignment refuses when it would strand self-imports", func(t *testing.T) {
		dir := repairTree(t, map[string]string{
			"cue.mod/module.cue": repairCueMod,
			"module.cue": `package app

import id "example.com/modules/app/identity"

_p: id.ModulePath
`,
			"identity/identity.cue": repairIdentity,
		})
		_, err := DetectRepair(ctx, nil, "", dir, "example.com/modules/renamed@v1")
		var refusalErr *RefusalError
		require.ErrorAs(t, err, &refusalErr)
		assert.Contains(t, refusalErr.Refusal.Headline, "strand self-imports")
	})

	t.Run("missing identity with no loadable version refuses — never chosen", func(t *testing.T) {
		dir := repairTree(t, map[string]string{
			"cue.mod/module.cue": repairCueMod,
		})
		_, err := DetectRepair(ctx, nil, "", dir, "")
		var refusalErr *RefusalError
		require.ErrorAs(t, err, &refusalErr)
		assert.Contains(t, refusalErr.Refusal.Headline, "states no version")
	})

	t.Run("malformed cue.mod is not silently rewritten", func(t *testing.T) {
		dir := repairTree(t, map[string]string{
			"cue.mod/module.cue":    "language: { version: \"v0.17.0\" }\n",
			"identity/identity.cue": repairIdentity,
		})
		_, err := DetectRepair(ctx, nil, "", dir, "")
		require.Error(t, err)
		assert.NotErrorAs(t, err, new(*RefusalError))
	})
}

func TestRepairPlanDescribe(t *testing.T) {
	dir := repairTree(t, map[string]string{
		"cue.mod/module.cue":    repairCueMod,
		"identity/identity.cue": repairIdentity,
	})
	plan, err := DetectRepair(context.Background(), nil, "", dir, "example.com/modules/renamed@v1")
	require.NoError(t, err)

	out := plan.Describe()
	// Aligned current → replacement pairs for every edit (D20: the author
	// must have something to judge).
	assert.Contains(t, out, "example.com/modules/renamed@v1")
	assert.Contains(t, out, "example.com/modules/app@v1 -> example.com/modules/renamed@v1")
	assert.Contains(t, out, filepath.Join("cue.mod", "module.cue"))
	assert.Contains(t, out, filepath.Join("identity", "identity.cue"))
}

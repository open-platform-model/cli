package scaffold

import (
	"context"
	"fmt"
	"testing"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// derivesTree builds a kind "Module" tree whose metadata states the given
// modulePath and version as literals; no imports, so the acquire needs no
// registry.
func derivesTree(t *testing.T, modulePath, version string) string {
	t.Helper()
	return repairTree(t, map[string]string{
		"cue.mod/module.cue": repairCueMod,
		"module.cue": fmt.Sprintf(`package app

kind: "Module"

metadata: {
	name:       "app"
	modulePath: %q
	version:    %q
}
`, modulePath, version),
	})
}

func TestAssertDerives(t *testing.T) {
	ctx := context.Background()
	const newPath = "example.com/modules/renamed@v1"

	t.Run("metadata matching the new identity passes", func(t *testing.T) {
		dir := derivesTree(t, newPath, InitialVersion)
		require.NoError(t, assertDerives(ctx, kernel.New(), dir, newPath))
	})

	cases := []struct {
		name, modulePath, version, field, derive string
	}{
		{"wrong modulePath refuses on modulePath", "example.com/modules/app@v1", InitialVersion, "modulePath", "id.ModulePath"},
		{"wrong version refuses on version", newPath, "1.2.0", "version", "id.Version"},
		{"both wrong refuses on modulePath", "example.com/modules/app@v1", "1.2.0", "modulePath", "id.ModulePath"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := derivesTree(t, tc.modulePath, tc.version)
			err := assertDerives(ctx, kernel.New(), dir, newPath)
			var refusalErr *RefusalError
			require.ErrorAs(t, err, &refusalErr)
			assert.Contains(t, refusalErr.Refusal.Headline, "does not derive metadata."+tc.field+" ")
			assert.Contains(t, refusalErr.Refusal.Action, tc.field+": "+tc.derive)
		})
	}
}

package core_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/core"
)

func TestComputeReleaseUUID(t *testing.T) {
	t.Run("same inputs produce same UUID", func(t *testing.T) {
		a := core.ComputeReleaseUUID("example.com/app@v1#app", "my-app", "production")
		b := core.ComputeReleaseUUID("example.com/app@v1#app", "my-app", "production")
		assert.Equal(t, a, b)
	})

	t.Run("different name produces different UUID", func(t *testing.T) {
		a := core.ComputeReleaseUUID("example.com/app@v1#app", "my-app", "production")
		b := core.ComputeReleaseUUID("example.com/app@v1#app", "other-app", "production")
		assert.NotEqual(t, a, b)
	})

	t.Run("different namespace produces different UUID", func(t *testing.T) {
		a := core.ComputeReleaseUUID("example.com/app@v1#app", "my-app", "production")
		b := core.ComputeReleaseUUID("example.com/app@v1#app", "my-app", "staging")
		assert.NotEqual(t, a, b)
	})

	t.Run("different fqn produces different UUID", func(t *testing.T) {
		a := core.ComputeReleaseUUID("example.com/app@v1#app", "my-app", "production")
		b := core.ComputeReleaseUUID("example.com/other@v1#other", "my-app", "production")
		assert.NotEqual(t, a, b)
	})

	t.Run("result is a valid UUID v5", func(t *testing.T) {
		result := core.ComputeReleaseUUID("example.com/app@v1#app", "my-app", "production")
		// UUID v5: version nibble at position 14 is '5'
		require.Len(t, result, 36)
		parts := strings.Split(result, "-")
		require.Len(t, parts, 5)
		assert.Equal(t, "5", string(parts[2][0]), "UUID version nibble should be 5")
	})

	t.Run("result is valid UUID format", func(t *testing.T) {
		result := core.ComputeReleaseUUID("example.com/app@v1#app", "my-app", "production")
		// Format: 8-4-4-4-12
		parts := strings.Split(result, "-")
		require.Len(t, parts, 5)
		assert.Len(t, parts[0], 8)
		assert.Len(t, parts[1], 4)
		assert.Len(t, parts[2], 4)
		assert.Len(t, parts[3], 4)
		assert.Len(t, parts[4], 12)
	})
}

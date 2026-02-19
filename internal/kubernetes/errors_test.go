package kubernetes

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoResourcesFoundError(t *testing.T) {
	t.Run("error message with release name", func(t *testing.T) {
		err := &noResourcesFoundError{
			ReleaseName: "my-app",
			Namespace:   "production",
		}
		assert.Equal(t, `no resources found for release my-app in namespace production`, err.Error())
	})

	t.Run("error message with release-id", func(t *testing.T) {
		err := &noResourcesFoundError{
			ReleaseID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Namespace: "production",
		}
		assert.Equal(t, `no resources found for release-id a1b2c3d4-e5f6-7890-abcd-ef1234567890 in namespace production`, err.Error())
	})

	t.Run("errors.Is matches errNoResourcesFound", func(t *testing.T) {
		err := &noResourcesFoundError{
			ReleaseName: "my-app",
			Namespace:   "production",
		}
		assert.True(t, errors.Is(err, errNoResourcesFound))
	})

	t.Run("IsNoResourcesFound matches direct error", func(t *testing.T) {
		err := &noResourcesFoundError{
			ReleaseName: "my-app",
			Namespace:   "production",
		}
		assert.True(t, IsNoResourcesFound(err))
	})

	t.Run("IsNoResourcesFound matches wrapped error", func(t *testing.T) {
		inner := &noResourcesFoundError{
			ReleaseName: "my-app",
			Namespace:   "production",
		}
		wrapped := fmt.Errorf("discovering release resources: %w", inner)
		assert.True(t, IsNoResourcesFound(wrapped))
	})

	t.Run("IsNoResourcesFound rejects unrelated error", func(t *testing.T) {
		err := fmt.Errorf("connection refused")
		assert.False(t, IsNoResourcesFound(err))
	})
}

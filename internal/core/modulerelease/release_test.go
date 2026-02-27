package modulerelease

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/core/component"
	opmerrors "github.com/opmodel/cli/internal/errors"
)

func TestValidate_ModuleRelease(t *testing.T) {
	t.Run("empty components map passes", func(t *testing.T) {
		rel := &ModuleRelease{Components: map[string]*component.Component{}}
		assert.NoError(t, rel.Validate())
	})

	t.Run("nil components map passes", func(t *testing.T) {
		rel := &ModuleRelease{}
		assert.NoError(t, rel.Validate())
	})

	t.Run("concrete component passes", func(t *testing.T) {
		ctx := cuecontext.New()
		concVal := ctx.CompileString(`{ x: 42 }`)
		require.NoError(t, concVal.Err())

		rel := &ModuleRelease{
			Components: map[string]*component.Component{
				"web": {Value: concVal},
			},
		}
		assert.NoError(t, rel.Validate())
	})

	t.Run("non-concrete component fails", func(t *testing.T) {
		ctx := cuecontext.New()
		openVal := ctx.CompileString(`{ x: int }`)
		require.NoError(t, openVal.Err())

		rel := &ModuleRelease{
			Components: map[string]*component.Component{
				"web": {Value: openVal},
			},
		}
		err := rel.Validate()
		require.Error(t, err)

		var valErr *opmerrors.ValidationError
		require.ErrorAs(t, err, &valErr)
		assert.Contains(t, valErr.Message, "non-concrete values")
	})
}

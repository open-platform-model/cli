package publish

import (
	"context"
	"errors"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	liberrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/kernel"
)

// realIdentitySchema loads core's #IdentityPackage from the registry, as
// realtree_test.go does. It skips only when the registry gives no response;
// any other failure to load the schema fails the test.
func realIdentitySchema(t *testing.T) (*kernel.Kernel, cue.Value) {
	t.Helper()
	k := kernel.New(kernel.WithRegistry(realTreeRegistry))
	schemaVal, err := k.SchemaCache().Get()
	if err != nil {
		var fetchErr *liberrors.FetchError
		if errors.As(liberrors.Classify(err), &fetchErr) && fetchErr.Kind == liberrors.FetchUnreachable {
			t.Skipf("registry unreachable, core schema not loaded: %v", err)
		}
		t.Fatalf("loading core schema: %v", err)
	}
	def := schemaVal.LookupPath(cue.MakePath(cue.Def("IdentityPackage")))
	require.True(t, def.Exists(), "core has #IdentityPackage")
	return k, def
}

// TestRealSchema_OpenVersion runs the open and fill cases against core's real
// #IdentityPackage, so the stub cannot drift from core's VersionMajor
// derivation unnoticed.
func TestRealSchema_OpenVersion(t *testing.T) {
	k, identitySchema := realIdentitySchema(t)

	// Only the schema comes from the registry. The already-published lookup
	// goes to an empty in-process registry.
	withRealSchema := func(flag string) func(*Options) {
		return func(o *Options) {
			o.Context = identitySchema.Context() //nolint:staticcheck // SA1019: the pipeline builds in the schema's runtime, as RunPublish does
			o.Kernel = k
			o.IdentitySchema = identitySchema
			o.VersionFlag = flag
		}
	}

	t.Run("vet passes an open Version", func(t *testing.T) {
		dir := writeTree(t, v0ModuleFiles("string"))
		p, _, err := VetChecks(context.Background(), Options{
			Dir:            dir,
			Kind:           KindModule,
			Context:        identitySchema.Context(), //nolint:staticcheck // SA1019: as above
			Kernel:         k,
			IdentitySchema: identitySchema,
		})
		require.NoError(t, err)
		assert.Empty(t, p.Refusals, refusalHeadlines(p))
	})

	t.Run("an open Version filled by the flag passes conformance", func(t *testing.T) {
		p := runFixture(t, KindModule, v0ModuleFiles("string"), withRealSchema("0.1.0"))
		require.Empty(t, p.Refusals, "%s\n%s", refusalHeadlines(p), allRefusalText(p))
		ver := p.identityField("Version")
		require.NotNil(t, ver)
		assert.Equal(t, StateOpen, ver.State)
		assert.True(t, ver.Filled)
		assert.Equal(t, "0.1.0", p.FillVersion)
		assert.Equal(t, "--version (fills the open Version)", p.TagSource)
	})

	t.Run("a filled Version is checked against the path major", func(t *testing.T) {
		p := runFixture(t, KindModule, v0ModuleFiles("string"), withRealSchema("1.0.0"))
		require.NotEmpty(t, p.Refusals)
		assert.False(t, p.Go(), "nothing is pushed")
		text := allRefusalText(p)
		assert.Contains(t, text, "v1")
		assert.Contains(t, text, "v0")
		if r := conformanceRefusal(p); r != nil {
			assert.Contains(t, r.Err.Error(), `"v1"`, "the identity check did not see the fill")
			assert.Contains(t, r.Err.Error(), `"v0"`, "the identity check did not see the fill")
		}
	})
}

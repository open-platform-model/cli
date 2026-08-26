package publish

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/schema"
)

// realTreeRegistry is the GHCR-first read mapping: opmodel.dev resolves from
// the public registry, everything else from the CUE default. Reads only —
// nothing here writes.
const realTreeRegistry = "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"

// TestRealTree_CatalogOpm is the registry-backed smoke over the real first
// -party catalog: the workspace's catalog_opm/opm must pass the member and
// posture gates AS-IS against the real core v2 schema, and the compat gate
// must run clean against the live GHCR history.
//
// KNOWN BOUND (documented, deliberate): the predecessor scan probes by the
// D49 filing convention (<kind>/<apiVersion> packages). The published history
// below the alpha releases includes pre-D49 `-0.dev.*` builds whose trees the
// convention cannot see into — probes there return the scan's negative
// signal and the walk continues. Acceptable: that horizon is entirely
// alpha-era, and alpha promises nothing (D34).
func TestRealTree_CatalogOpm(t *testing.T) {
	src, err := filepath.Abs(filepath.Join("..", "..", "..", "catalog_opm", "opm"))
	require.NoError(t, err)
	if _, err := os.Stat(src); err != nil {
		t.Skip("catalog_opm workspace checkout not present beside cli/")
	}

	k := kernel.New(
		kernel.WithRegistry(realTreeRegistry),
		kernel.WithSchemaLoader(schema.OCILoader{Registry: realTreeRegistry}),
	)
	schemaVal, err := k.SchemaCache().Get(k.CueContext())
	if err != nil {
		t.Skipf("core v2 schema unavailable (registry/cache): %v", err)
	}

	opts := Options{
		Dir:                     src,
		Kind:                    KindCatalog,
		Context:                 k.CueContext(),
		IdentitySchema:          schemaVal.LookupPath(cue.MakePath(cue.Def("IdentityPackage"))),
		MemberFQNGateSchema:     schemaVal.LookupPath(cue.MakePath(cue.Def("CatalogMemberFQNGate"))),
		TraitOptionalGateSchema: schemaVal.LookupPath(cue.MakePath(cue.Def("TraitOptionalGate"))),
		Registry:                realTreeRegistry,
	}
	require.True(t, opts.IdentitySchema.Exists())
	require.True(t, opts.MemberFQNGateSchema.Exists())
	require.True(t, opts.TraitOptionalGateSchema.Exists())

	a, refusal := loadArtifact(opts)
	require.Nil(t, refusal, "catalog_opm/opm must load")

	members, refusals := enumerateMembers(opts, src)
	require.Empty(t, refusals)

	p := &Plan{Kind: KindCatalog, Dir: src}
	gateMemberFQN(p, opts, a, members)
	gateTraitOptional(p, opts, members)
	require.Empty(t, p.Refusals, "catalog_opm/opm must pass the member and posture gates as-is:\n%s\n%s",
		refusalHeadlines(p), refusalErrors(p))

	// Order-of-magnitude sanity, not exact counts — the catalog grows.
	assert.GreaterOrEqual(t, p.CatalogGates.MembersChecked, 50)
	assert.GreaterOrEqual(t, p.CatalogGates.TraitsChecked, 10)

	// The compat gate against the live published history.
	modPath, err := a.identity.LookupPath(cue.ParsePath("ModulePath")).String()
	require.NoError(t, err)
	version, err := a.identity.LookupPath(cue.ParsePath("Version")).String()
	require.NoError(t, err)
	repo, major, ok := strings.Cut(modPath, "@")
	require.True(t, ok)

	client, err := NewRegistryClient(realTreeRegistry)
	require.NoError(t, err)
	versions, err := client.ModuleVersions(context.Background(), modPath)
	if err != nil {
		t.Skipf("GHCR unreachable for version enumeration: %v", err)
	}
	preds := predecessorVersions(versions, "v"+version, major)
	require.NotEmpty(t, preds, "the live history should hold at least one predecessor build")

	if err := compatScan(opts, repo, preds, members, &p.CatalogGates, p.refuse); err != nil {
		var connErr *ConnectivityError
		if errors.As(err, &connErr) {
			t.Skipf("GHCR unreachable mid-walk: %v", err)
		}
		t.Fatalf("compat walk failed: %v", err)
	}
	assert.Empty(t, p.Refusals, "the live history must compare clean:\n%s\n%s",
		refusalHeadlines(p), refusalDetails(p))
	assert.Greater(t, p.CatalogGates.CompatCompared, 0)
	t.Logf("real-tree outcome: %+v (predecessors: %v)", p.CatalogGates, preds)
}

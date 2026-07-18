package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGateCRDPresent(t *testing.T) {
	ctx := context.Background()

	t.Run("present passes", func(t *testing.T) {
		client := newDynamicClient(makeModuleInstanceCRD(true, true))
		require.NoError(t, GateCRDPresent(ctx, client))
	})

	t.Run("absent yields install hint", func(t *testing.T) {
		client := newDynamicClient()
		err := GateCRDPresent(ctx, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ModuleInstance CRD not found")
		assert.Contains(t, err.Error(), "opm operator install --crds-only")
	})
}

func TestGateCRDFieldFloor(t *testing.T) {
	ctx := context.Background()

	t.Run("both fields present passes", func(t *testing.T) {
		client := newDynamicClient(makeModuleInstanceCRD(true, true))
		require.NoError(t, GateCRDFieldFloor(ctx, client))
	})

	t.Run("missing spec.owner refused", func(t *testing.T) {
		client := newDynamicClient(makeModuleInstanceCRD(false, true))
		err := GateCRDFieldFloor(ctx, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required fields")
	})

	t.Run("missing status.inventory refused", func(t *testing.T) {
		client := newDynamicClient(makeModuleInstanceCRD(true, false))
		err := GateCRDFieldFloor(ctx, client)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required fields")
	})
}

func TestGateOperatorVersionCeiling(t *testing.T) {
	ctx := context.Background()

	t.Run("dev build skips", func(t *testing.T) {
		client := newDynamicClient(makePlatform("2.0.0"))
		require.NoError(t, GateOperatorVersionCeiling(ctx, client, "dev"))
	})

	t.Run("non-semver CLI skips", func(t *testing.T) {
		client := newDynamicClient(makePlatform("2.0.0"))
		require.NoError(t, GateOperatorVersionCeiling(ctx, client, "not-a-version"))
	})

	t.Run("no Platform skips (solo cluster)", func(t *testing.T) {
		client := newDynamicClient()
		require.NoError(t, GateOperatorVersionCeiling(ctx, client, "1.0.0"))
	})

	t.Run("absent operatorVersion skips", func(t *testing.T) {
		client := newDynamicClient(makePlatform(""))
		require.NoError(t, GateOperatorVersionCeiling(ctx, client, "1.0.0"))
	})

	t.Run("newer operator refuses older CLI", func(t *testing.T) {
		client := newDynamicClient(makePlatform("1.2.0"))
		err := GateOperatorVersionCeiling(ctx, client, "1.1.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "older than the cluster operator")
	})

	t.Run("equal version passes", func(t *testing.T) {
		client := newDynamicClient(makePlatform("1.2.0"))
		require.NoError(t, GateOperatorVersionCeiling(ctx, client, "1.2.0"))
	})

	t.Run("tolerates v-prefixed versions on both sides", func(t *testing.T) {
		client := newDynamicClient(makePlatform("v1.2.0"))
		require.NoError(t, GateOperatorVersionCeiling(ctx, client, "v1.3.0"))
	})
}

func TestGateStatusRBAC(t *testing.T) {
	ctx := context.Background()

	t.Run("allowed passes silently", func(t *testing.T) {
		client := withSSAR(newDynamicClient(), true)
		require.NoError(t, GateStatusRBAC(ctx, client, "demo"))
	})

	t.Run("denied aborts with rbac hint", func(t *testing.T) {
		client := withSSAR(newDynamicClient(), false)
		err := GateStatusRBAC(ctx, client, "demo")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "moduleinstances/status")
		assert.Contains(t, err.Error(), "--rbac")
	})
}

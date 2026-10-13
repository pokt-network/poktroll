package upgrades_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/upgrades"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
)

func TestUpgrade_NEXT_SeedsServiceCapParams(t *testing.T) {
	k, ctx := testkeeper.SupplierKeeper(t)

	// Simulate a pre-upgrade chain: params without the new fields.
	params := k.Keeper.GetParams(ctx)
	params.MinStakeIncludedServices = 0
	params.StakePerAdditionalService = nil
	require.NoError(t, k.SetParams(ctx, params))

	logger := cosmostypes.UnwrapSDKContext(ctx).Logger()
	require.NoError(t, upgrades.SeedServiceCapParams(ctx, logger, *k.Keeper))

	seeded := k.Keeper.GetParams(ctx)
	require.Equal(t, uint64(1), seeded.MinStakeIncludedServices)
	require.True(t, seeded.StakePerAdditionalService.IsZero())
	require.False(t, seeded.ServiceCapEnabled())
	require.NoError(t, seeded.Validate())
	require.False(t, k.IsServiceCapSweepPending(ctx))
}

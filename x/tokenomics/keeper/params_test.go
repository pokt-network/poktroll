package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.TokenomicsKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.ComputeToTokensMultiplier, k.ComputeToTokensMultiplier(ctx))
}

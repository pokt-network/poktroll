package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "pocket/testutil/keeper"
	"pocket/x/supplier/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.SupplierKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.EarlietClaimSubmissionOffset, k.EarlietClaimSubmissionOffset(ctx))
	require.EqualValues(t, params.EarliestProofSubmissionOffset, k.EarliestProofSubmissionOffset(ctx))
	require.EqualValues(t, params.LatestClaimSubmissionBlocksInterval, k.LatestClaimSubmissionBlocksInterval(ctx))
	require.EqualValues(t, params.LatestProofSubmissionBlocksInterval, k.LatestProofSubmissionBlocksInterval(ctx))
	require.EqualValues(t, params.ClaimSubmissionBlocksWindow, k.ClaimSubmissionBlocksWindow(ctx))
	require.EqualValues(t, params.ProofSubmissionBlocksWindow, k.ProofSubmissionBlocksWindow(ctx))
}

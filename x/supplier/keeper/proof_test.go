package keeper_test

import (
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNProofs(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Proof {
	proofs := make([]types.Proof, n)
	for i := range proofs {
		proofs[i] = types.Proof{
			SupplierAddress: sample.AccAddress(),
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress: sample.AccAddress(),
				Service:            &sharedtypes.Service{Id: testServiceId},
				SessionId:          fmt.Sprintf("session-%d", i),
				// TODO_IN_THIS_COMMIT: increment start/end height using i.
				SessionStartBlockHeight: 1,
				SessionEndBlockHeight:   1 + sessionkeeper.NumBlocksPerSession,
			},
			MerkleProof: nil,
		}

		keeper.UpsertProof(ctx, proofs[i])
	}
	return proofs
}

func TestProofGet(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	proofs := createNProofs(keeper, ctx, 10)
	for _, proof := range proofs {
		rst, found := keeper.GetProof(ctx,
			proof.GetSessionHeader().GetSessionId(),
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&proof),
			nullify.Fill(&rst),
		)
	}
}
func TestProofRemove(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	items := createNProofs(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveProof(ctx,
			item.GetSessionHeader().GetSessionId(),
		)
		_, found := keeper.GetProof(ctx,
			item.GetSessionHeader().GetSessionId(),
		)
		require.False(t, found)
	}
}

func TestProofGetAll(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	items := createNProofs(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllProofs(ctx)),
	)
}

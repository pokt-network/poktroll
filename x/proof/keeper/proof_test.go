package keeper_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	"github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	testServiceId = "svc1"
	testSessionId = "mock_session_id"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNProofs(keeper keeper.Keeper, ctx context.Context, n int) []types.Proof {
	proofs := make([]types.Proof, n)

	for i := range proofs {
		proofs[i] = types.Proof{
			SupplierAddress: sample.AccAddress(),
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress:      sample.AccAddress(),
				Service:                 &sharedtypes.Service{Id: testServiceId},
				SessionId:               fmt.Sprintf("session-%d", i),
				SessionStartBlockHeight: 1,
				SessionEndBlockHeight:   sessionkeeper.GetSessionEndBlockHeight(1),
			},
			ClosestMerkleProof: nil,
		}

		keeper.UpsertProof(ctx, proofs[i])
	}

	return proofs
}

func TestProofGet(t *testing.T) {
	keeper, ctx := keepertest.ProofKeeper(t)
	proofs := createNProofs(keeper, ctx, 10)

	for _, proof := range proofs {
		foundProof, isProofFound := keeper.GetProof(
			ctx,
			proof.GetSessionHeader().GetSessionId(),
			proof.GetSupplierAddress(),
		)
		require.True(t, isProofFound)
		require.Equal(t,
			nullify.Fill(&proof),
			nullify.Fill(&foundProof),
		)
	}
}
func TestProofRemove(t *testing.T) {
	keeper, ctx := keepertest.ProofKeeper(t)
	proofs := createNProofs(keeper, ctx, 10)
	for _, proof := range proofs {
		sessionId := proof.GetSessionHeader().GetSessionId()
		keeper.RemoveProof(ctx, sessionId, proof.GetSupplierAddress())
		_, isProofFound := keeper.GetProof(ctx, sessionId, proof.GetSupplierAddress())
		require.False(t, isProofFound)
	}
}

func TestProofGetAll(t *testing.T) {
	keeper, ctx := keepertest.ProofKeeper(t)
	proofs := createNProofs(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(proofs),
		nullify.Fill(keeper.GetAllProofs(ctx)),
	)
}

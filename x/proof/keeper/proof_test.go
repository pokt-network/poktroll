package keeper_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/pocket/testutil/keeper"
	"github.com/pokt-network/pocket/testutil/nullify"
	"github.com/pokt-network/pocket/testutil/sample"
	testsession "github.com/pokt-network/pocket/testutil/session"
	"github.com/pokt-network/pocket/x/proof/keeper"
	"github.com/pokt-network/pocket/x/proof/types"
	sessiontypes "github.com/pokt-network/pocket/x/session/types"
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
			SupplierOperatorAddress: sample.AccAddress(),
			SessionHeader: &sessiontypes.SessionHeader{
				ApplicationAddress:      sample.AccAddress(),
				ServiceId:               testServiceId,
				SessionId:               fmt.Sprintf("session-%d", i),
				SessionStartBlockHeight: 1,
				SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
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
			proof.GetSupplierOperatorAddress(),
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
		keeper.RemoveProof(ctx, sessionId, proof.GetSupplierOperatorAddress())
		_, isProofFound := keeper.GetProof(ctx, sessionId, proof.GetSupplierOperatorAddress())
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

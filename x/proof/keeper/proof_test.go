package keeper_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/proof"
	"github.com/pokt-network/poktroll/proto/types/session"
	"github.com/pokt-network/poktroll/proto/types/shared"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	"github.com/pokt-network/poktroll/x/proof/keeper"
)

const (
	testServiceId = "svc1"
	testSessionId = "mock_session_id"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNProofs(keeper keeper.Keeper, ctx context.Context, n int) []proof.Proof {
	proofs := make([]proof.Proof, n)

	for i := range proofs {
		proofs[i] = proof.Proof{
			SupplierAddress: sample.AccAddress(),
			SessionHeader: &session.SessionHeader{
				ApplicationAddress:      sample.AccAddress(),
				Service:                 &shared.Service{Id: testServiceId},
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

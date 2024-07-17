package keeper_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/proof"
	sessiontypes "github.com/pokt-network/poktroll/proto/types/session"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/proof/keeper"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNClaims(keeper keeper.Keeper, ctx context.Context, n int) []proof.Claim {
	claims := make([]proof.Claim, n)

	for i := range claims {
		claims[i].SupplierAddress = sample.AccAddress()
		claims[i].SessionHeader = &sessiontypes.SessionHeader{
			SessionId:             fmt.Sprintf("session-%d", i),
			SessionEndBlockHeight: int64(i),
		}
		claims[i].RootHash = []byte(fmt.Sprintf("rootHash-%d", i))
		keeper.UpsertClaim(ctx, claims[i])
	}

	return claims
}

func TestClaimGet(t *testing.T) {
	keeper, ctx := keepertest.ProofKeeper(t)
	claims := createNClaims(keeper, ctx, 10)

	for _, claim := range claims {
		foundClaim, isClaimFound := keeper.GetClaim(
			ctx,
			claim.GetSessionHeader().GetSessionId(),
			claim.SupplierAddress,
		)
		require.True(t, isClaimFound)
		require.Equal(t,
			nullify.Fill(&claim),
			nullify.Fill(&foundClaim),
		)
	}
}
func TestClaimRemove(t *testing.T) {
	keeper, ctx := keepertest.ProofKeeper(t)
	claims := createNClaims(keeper, ctx, 10)

	for _, claim := range claims {
		sessionId := claim.GetSessionHeader().GetSessionId()
		keeper.RemoveClaim(ctx, sessionId, claim.SupplierAddress)
		_, isClaimFound := keeper.GetClaim(ctx, sessionId, claim.SupplierAddress)
		require.False(t, isClaimFound)
	}
}

func TestClaimGetAll(t *testing.T) {
	keeper, ctx := keepertest.ProofKeeper(t)
	claims := createNClaims(keeper, ctx, 10)

	// Get all the claims and check if they match
	allFoundClaims := keeper.GetAllClaims(ctx)
	require.ElementsMatch(t,
		nullify.Fill(claims),
		nullify.Fill(allFoundClaims),
	)
}

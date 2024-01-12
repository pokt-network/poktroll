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
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNClaims(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Claim {
	claims := make([]types.Claim, n)
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

func TestClaim_Get(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	claims := createNClaims(keeper, ctx, 10)
	for _, claim := range claims {
		foundClaim, isClaimFound := keeper.GetClaim(ctx,
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

func TestClaim_Remove(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	claims := createNClaims(keeper, ctx, 10)
	for _, claim := range claims {
		sessionId := claim.GetSessionHeader().GetSessionId()
		keeper.RemoveClaim(ctx,
			sessionId,
			claim.SupplierAddress,
		)
		_, isClaimFound := keeper.GetClaim(ctx,
			sessionId,
			claim.SupplierAddress,
		)
		require.False(t, isClaimFound)
	}
}

func TestClaim_GetAll(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	claims := createNClaims(keeper, ctx, 10)

	// Get all the claims and check if they match
	allFoundClaims := keeper.GetAllClaims(ctx)
	require.ElementsMatch(t,
		nullify.Fill(claims),
		nullify.Fill(allFoundClaims),
	)
}

func TestClaim_GetAll_ByAddress(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	claims := createNClaims(keeper, ctx, 10)

	// Get all claims for a given address
	allFoundClaimsByAddress := keeper.GetClaimsByAddress(ctx, claims[3].SupplierAddress)
	require.ElementsMatch(t,
		nullify.Fill([]types.Claim{claims[3]}),
		nullify.Fill(allFoundClaimsByAddress),
	)
}

func TestClaim_GetAll_ByHeight(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	claims := createNClaims(keeper, ctx, 10)

	// Get all claims for a given ending session block height
	sessionEndHeight := claims[6].GetSessionHeader().GetSessionEndBlockHeight()
	allFoundClaimsEndingAtHeight := keeper.GetClaimsByHeight(ctx, uint64(sessionEndHeight))
	require.ElementsMatch(t,
		nullify.Fill([]types.Claim{claims[6]}),
		nullify.Fill(allFoundClaimsEndingAtHeight),
	)
}

func TestClaim_GetAll_BySession(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	claims := createNClaims(keeper, ctx, 10)

	// Get all claims for a given ending session block height
	sessionId := claims[8].GetSessionHeader().GetSessionId()
	allFoundClaimsForSession := keeper.GetClaimsBySession(ctx, sessionId)
	require.ElementsMatch(t,
		nullify.Fill([]types.Claim{claims[8]}),
		nullify.Fill(allFoundClaimsForSession),
	)
}

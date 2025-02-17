package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestMorseAccountClaimMsgServerCreate(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	for i := 0; i < 5; i++ {
		msgClaim := &types.MsgClaimMorseAccount{
			ShannonDestAddress: sample.AccAddress(),
			MorseSrcAddress:    sample.MorseAddressHex(),
		}
		_, err := srv.ClaimMorseAccount(ctx, msgClaim)
		require.NoError(t, err)
		rst, found := k.GetMorseClaimableAccount(ctx,
			msgClaim.MorseSrcAddress,
		)
		require.True(t, found)

		expectedRes := &types.MsgClaimMorseAccountResponse{
			MorseSrcAddress: msgClaim.MorseSrcAddress,
			ClaimedBalance:  sdk.NewInt64Coin(volatile.DenomuPOKT, 0),
			ClaimedAtHeight: 0,
		}
		require.Equal(t, expectedRes, rst)

		// assert each event was emitted...
		// assert that the morse account was clamed...
	}
}

// TODO_IN_THIS_COMMIT: error cases...
// - invalid ValidateBasic()
// - MorseClaimableAccount not found

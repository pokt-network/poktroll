package keeper_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestMorseAccountClaimMsgServerCreate(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	shannonDestAddress := "A"
	for i := 0; i < 5; i++ {
		expected := &types.MsgCreateMorseAccountClaim{ShannonDestAddress: shannonDestAddress,
			MorseSrcAddress: strconv.Itoa(i),
		}
		_, err := srv.CreateMorseAccountClaim(ctx, expected)
		require.NoError(t, err)
		rst, found := k.GetMorseAccountClaim(ctx,
			expected.MorseSrcAddress,
		)
		require.True(t, found)
		require.Equal(t, expected.ShannonDestAddress, rst.ShannonDestAddress)
	}
}

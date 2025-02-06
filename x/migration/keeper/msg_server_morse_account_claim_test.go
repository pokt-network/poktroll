package keeper_test

import (
	"encoding/hex"
	rand2 "math/rand/v2"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
)

// TODO_IMPROVE: Promote this to a shared testutil pkg.
const morseAddressLengthBytes = 20

var (
	// Prevent strconv unused error
	_   = strconv.IntSize
	rng = rand2.NewChaCha8([32]byte{})
)

func TestMorseAccountClaimMsgServerCreate(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	for i := 0; i < 5; i++ {
		expected := &types.MsgCreateMorseAccountClaim{
			ShannonDestAddress: sample.AccAddress(),
			MorseSrcAddress:    randomMorseAddressBytes(t),
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

// randomMorseAddressBytes returns a random hex-encoded string with the same
// length as a valid morse address.
func randomMorseAddressBytes(t *testing.T) string {
	addrBz := make([]byte, morseAddressLengthBytes)
	_, err := rng.Read(addrBz)
	require.NoError(t, err)

	return hex.EncodeToString(addrBz)
}

package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/migration/keeper"
	"github.com/pokt-network/poktroll/x/migration/types"
)

func TestMorseAccountStateMsgServerCreate(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	authority := sample.AccAddress()
	expected, err := types.NewMsgCreateMorseAccountState(authority, types.MorseAccountState{})
	require.NoError(t, err)

	_, err = srv.CreateMorseAccountState(ctx, expected)
	require.NoError(t, err)
	_, found := k.GetMorseAccountState(ctx)
	require.True(t, found)
}

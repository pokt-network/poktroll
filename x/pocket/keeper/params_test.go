package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "pocket/testutil/keeper"
	"pocket/x/pocket/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.PocketKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}

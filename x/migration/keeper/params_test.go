package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

    keepertest "github.com/pokt-network/poktroll/testutil/keeper"
    "github.com/pokt-network/poktroll/x/migration/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.MigrationKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}

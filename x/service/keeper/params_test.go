package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/service"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	params := service.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}

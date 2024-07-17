package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/session"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := testkeeper.SessionKeeper(t)
	params := session.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	response, err := keeper.Params(ctx, &session.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &session.QueryParamsResponse{Params: params}, response)
}

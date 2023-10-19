package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/sample"
	"pocket/x/session/types"
)

func TestSession_GetSession_Success(t *testing.T) {
	keeper, ctx := keepertest.SessionKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	req := &types.QueryGetSessionRequest{
		ApplicationAddress: sample.AccAddress(),
		ServiceId:          "service_id",
		BlockHeight:        1,
	}

	response, err := keeper.GetSession(wctx, req)
	require.NoError(t, err)
	require.Equal(t, response, nil)

}

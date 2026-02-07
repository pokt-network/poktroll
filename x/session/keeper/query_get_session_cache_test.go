package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/session/types"
)

func TestCachedQueryServer_GetSession_CacheHit(t *testing.T) {
	k, ctx := keepertest.SessionKeeper(t, sharedParamsOpt)
	ctx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(100)

	cachedServer := k.NewCachedQueryServer()

	req := &types.QueryGetSessionRequest{
		ApplicationAddress: keepertest.TestApp1Address,
		ServiceId:          keepertest.TestServiceId1,
		BlockHeight:        1,
	}

	// First call — cache miss, delegates to keeper.
	res1, err := cachedServer.GetSession(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res1)
	require.Equal(t, keepertest.TestApp1Address, res1.Session.Header.ApplicationAddress)

	// Second call — cache hit, should return identical result.
	res2, err := cachedServer.GetSession(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, res2)

	// Both responses should be the same pointer (cached).
	require.Same(t, res1, res2)
}

func TestCachedQueryServer_GetSession_SameSessionDifferentHeights(t *testing.T) {
	k, ctx := keepertest.SessionKeeper(t, sharedParamsOpt)
	ctx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(100)

	cachedServer := k.NewCachedQueryServer()

	// With NumBlocksPerSession=4, heights 1,2,3,4 are all in session starting at height 1.
	req1 := &types.QueryGetSessionRequest{
		ApplicationAddress: keepertest.TestApp1Address,
		ServiceId:          keepertest.TestServiceId1,
		BlockHeight:        1,
	}
	req2 := &types.QueryGetSessionRequest{
		ApplicationAddress: keepertest.TestApp1Address,
		ServiceId:          keepertest.TestServiceId1,
		BlockHeight:        3,
	}

	res1, err := cachedServer.GetSession(ctx, req1)
	require.NoError(t, err)
	require.NotNil(t, res1)

	// Different block height in the same session should yield a cache hit.
	res2, err := cachedServer.GetSession(ctx, req2)
	require.NoError(t, err)
	require.NotNil(t, res2)

	// Same cached pointer because both normalize to the same session start height.
	require.Same(t, res1, res2)
}

func TestCachedQueryServer_GetSession_DifferentServicesMiss(t *testing.T) {
	k, ctx := keepertest.SessionKeeper(t, sharedParamsOpt)
	ctx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(100)

	cachedServer := k.NewCachedQueryServer()

	req1 := &types.QueryGetSessionRequest{
		ApplicationAddress: keepertest.TestApp1Address,
		ServiceId:          keepertest.TestServiceId1,
		BlockHeight:        1,
	}
	req2 := &types.QueryGetSessionRequest{
		ApplicationAddress: keepertest.TestApp1Address,
		ServiceId:          keepertest.TestServiceId12,
		BlockHeight:        1,
	}

	res1, err := cachedServer.GetSession(ctx, req1)
	require.NoError(t, err)

	res2, err := cachedServer.GetSession(ctx, req2)
	require.NoError(t, err)

	// Different services should yield different cached results.
	require.NotSame(t, res1, res2)
	require.Equal(t, keepertest.TestServiceId1, res1.Session.Header.ServiceId)
	require.Equal(t, keepertest.TestServiceId12, res2.Session.Header.ServiceId)
}

func TestCachedQueryServer_GetSession_NilRequest(t *testing.T) {
	k, ctx := keepertest.SessionKeeper(t, sharedParamsOpt)
	ctx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(100)

	cachedServer := k.NewCachedQueryServer()

	// Nil request should pass through to keeper (which returns an error).
	res, err := cachedServer.GetSession(ctx, nil)
	require.Error(t, err)
	require.Nil(t, res)
}

func TestCachedQueryServer_GetSession_ErrorCachedReturnsSameError(t *testing.T) {
	k, ctx := keepertest.SessionKeeper(t, sharedParamsOpt)
	ctx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(100)

	cachedServer := k.NewCachedQueryServer()

	tests := []struct {
		desc string
		req  *types.QueryGetSessionRequest
	}{
		{
			desc: "app not found",
			req: &types.QueryGetSessionRequest{
				ApplicationAddress: "pokt1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				ServiceId:          keepertest.TestServiceId1,
				BlockHeight:        1,
			},
		},
		{
			desc: "no suppliers for service",
			req: &types.QueryGetSessionRequest{
				ApplicationAddress: keepertest.TestApp1Address,
				ServiceId:          keepertest.TestServiceId11,
				BlockHeight:        1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// First call — cache miss, error goes through keeper.
			res1, err1 := cachedServer.GetSession(ctx, tt.req)
			require.Error(t, err1)
			require.Nil(t, res1)

			// Second call — cache hit, must return the identical error.
			res2, err2 := cachedServer.GetSession(ctx, tt.req)
			require.Error(t, err2)
			require.Nil(t, res2)

			require.Equal(t, err1.Error(), err2.Error())
		})
	}
}

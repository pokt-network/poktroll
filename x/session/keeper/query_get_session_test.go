package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"pocket/cmd/pocketd/cmd"
	keepertest "pocket/testutil/keeper"
	"pocket/testutil/sample"
	"pocket/x/session/types"
	sharedtypes "pocket/x/shared/types"
)

func init() {
	cmd.InitSDKConfig()
}

// NOTE: See `session_hydrator_test.go` for more extensive test coverage of different
// GetSession scenarios. This is just used to verify a few basic scenarios that act as
// the Cosmos SDK context aware wrapper around it.

func TestSession_GetSession_Success(t *testing.T) {
	keeper, ctx := keepertest.SessionKeeper(t)
	ctx = ctx.WithBlockHeight(100) // provide a sufficiently large block height to avoid errors
	wctx := sdk.WrapSDKContext(ctx)

	type test struct {
		name string

		appAddr     string
		serviceId   string
		blockHeight int64

		expectedSessionId     string
		expectedSessionNumber int64
		expectedNumSuppliers  int
	}

	tests := []test{
		{
			name: "valid - app1 svc1 at height=1",

			appAddr:     keepertest.TestApp1Address,
			serviceId:   keepertest.TestServiceId1,
			blockHeight: 1,

			// Intentionally only checking a subset of the session metadata returned
			expectedSessionId:     "cf5bbdce56ee5a7c46c5d5482303907685a7e5dbb22703cd4a85df521b9ab6e9",
			expectedSessionNumber: 0,
			expectedNumSuppliers:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &types.QueryGetSessionRequest{
				ApplicationAddress: tt.appAddr,
				ServiceId: &sharedtypes.ServiceId{
					Id: tt.serviceId,
				},
				BlockHeight: 1,
			}

			response, err := keeper.GetSession(wctx, req)
			require.NoError(t, err)
			require.NotNil(t, response)

			require.Equal(t, tt.expectedSessionId, response.Session.SessionId)
			require.Equal(t, tt.expectedSessionNumber, response.Session.SessionNumber)
			require.Len(t, response.Session.Suppliers, tt.expectedNumSuppliers)
		})
	}
}

func TestSession_GetSession_Failure(t *testing.T) {
	keeper, ctx := keepertest.SessionKeeper(t)
	ctx = ctx.WithBlockHeight(100) // provide a sufficiently large block height to avoid errors
	wctx := sdk.WrapSDKContext(ctx)

	type test struct {
		name string

		appAddr     string
		serviceId   string
		blockHeight int64

		expectedErrContains string
	}

	tests := []test{
		{
			name: "application address does not reflected a staked application",

			appAddr:     sample.AccAddress(), // a random (valid) app address that's not staked
			serviceId:   keepertest.TestServiceId1,
			blockHeight: 1,

			expectedErrContains: types.ErrAppNotFound.Error(),
		},
		{
			name: "application staked for service that has no available suppliers",

			appAddr:     keepertest.TestApp1Address,
			serviceId:   keepertest.TestServiceId11,
			blockHeight: 1,

			expectedErrContains: types.ErrSuppliersNotFound.Error(),
		},
		{
			name: "application is valid but not staked for the specified service",

			appAddr:     keepertest.TestApp1Address,
			serviceId:   "svc9001", // App1 is not staked for service over 9000
			blockHeight: 1,

			expectedErrContains: types.ErrAppNotStakedForService.Error(),
		},
		{
			name: "application address is invalid format",

			appAddr:     "invalid_app_address",
			serviceId:   keepertest.TestServiceId1,
			blockHeight: 1,

			expectedErrContains: types.ErrSessionInvalidAppAddress.Error(),
		},
		{
			name: "service ID is invalid",

			appAddr:     keepertest.TestApp1Address,
			serviceId:   "invalid_serviceId",
			blockHeight: 1,

			expectedErrContains: "invalid serviceID for session being retrieved",
		},
		{
			name: "negative block height",

			appAddr:     keepertest.TestApp1Address,
			serviceId:   keepertest.TestServiceId1,
			blockHeight: -1,

			expectedErrContains: "invalid block height for session being retrieved",
		},
	}

	expectedRes := (*types.QueryGetSessionResponse)(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &types.QueryGetSessionRequest{
				ApplicationAddress: tt.appAddr,
				ServiceId: &sharedtypes.ServiceId{
					Id: tt.serviceId,
				},
				BlockHeight: tt.blockHeight,
			}

			res, err := keeper.GetSession(wctx, req)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedErrContains)
			require.Equal(t, expectedRes, res)
		})
	}
}

package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/proto/types/session"
	"github.com/pokt-network/poktroll/proto/types/shared"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
)

func init() {
	cmd.InitSDKConfig()
}

// NOTE: See `session_hydrator_test.go` for more extensive test coverage of different
// GetSession scenarios. This is just used to verify a few basic scenarios that act as
// the Cosmos SDK context aware wrapper around it.

func TestSession_GetSession_Success(t *testing.T) {
	keeper, ctx := keepertest.SessionKeeper(t)
	ctx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(100) // provide a sufficiently large block height to avoid errors

	tests := []struct {
		desc string

		appAddr     string
		serviceId   string
		blockHeight int64

		expectedSessionId     string
		expectedSessionNumber int64
		expectedNumSuppliers  int
	}{
		{
			desc: "valid - app1 svc1 at height=1",

			appAddr:     keepertest.TestApp1Address,
			serviceId:   keepertest.TestServiceId1,
			blockHeight: 1,

			// Intentionally only checking a subset of the session metadata returned
			expectedSessionId:     "afd00273055a4fddc0beb30074e14d474c07f9e895d2795db006a5139048d54b",
			expectedSessionNumber: 1,
			expectedNumSuppliers:  1,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			req := &session.QueryGetSessionRequest{
				ApplicationAddress: test.appAddr,
				Service: &shared.Service{
					Id: test.serviceId,
				},
				BlockHeight: 1,
			}

			response, err := keeper.GetSession(ctx, req)
			require.NoError(t, err)
			require.NotNil(t, response)

			require.Equal(t, test.expectedSessionId, response.Session.SessionId)
			require.Equal(t, test.expectedSessionNumber, response.Session.SessionNumber)
			require.Len(t, response.Session.Suppliers, test.expectedNumSuppliers)
		})
	}
}

func TestSession_GetSession_Failure(t *testing.T) {
	keeper, ctx := keepertest.SessionKeeper(t)
	ctx = sdk.UnwrapSDKContext(ctx).WithBlockHeight(100) // provide a sufficiently large block height to avoid errors

	tests := []struct {
		desc string

		appAddr     string
		serviceId   string
		blockHeight int64

		expectedErrMsg string
	}{
		{
			desc: "application address does not reflected a staked application",

			appAddr:     sample.AccAddress(), // a random (valid) app address that's not staked
			serviceId:   keepertest.TestServiceId1,
			blockHeight: 1,

			expectedErrMsg: session.ErrSessionAppNotFound.Error(),
		},
		{
			desc: "application staked for service that has no available suppliers",

			appAddr:     keepertest.TestApp1Address,
			serviceId:   keepertest.TestServiceId11,
			blockHeight: 1,

			expectedErrMsg: session.ErrSessionSuppliersNotFound.Error(),
		},
		{
			desc: "application is valid but not staked for the specified service",

			appAddr:     keepertest.TestApp1Address,
			serviceId:   "svc9001", // App1 is not staked for service over 9000
			blockHeight: 1,

			expectedErrMsg: session.ErrSessionAppNotStakedForService.Error(),
		},
		{
			desc: "application address is invalid format",

			appAddr:     "invalid_app_address",
			serviceId:   keepertest.TestServiceId1,
			blockHeight: 1,

			expectedErrMsg: session.ErrSessionInvalidAppAddress.Error(),
		},
		{
			desc: "service ID is invalid",

			appAddr:     keepertest.TestApp1Address,
			serviceId:   "service_id_is_too_long_to_be_valid",
			blockHeight: 1,

			expectedErrMsg: "invalid service in session",
		},
		{
			desc: "negative block height",

			appAddr:     keepertest.TestApp1Address,
			serviceId:   keepertest.TestServiceId1,
			blockHeight: -1,

			expectedErrMsg: "invalid block height for session being retrieved",
		},
	}

	expectedRes := (*session.QueryGetSessionResponse)(nil)

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			req := &session.QueryGetSessionRequest{
				ApplicationAddress: test.appAddr,
				Service: &shared.Service{
					Id: test.serviceId,
				},
				BlockHeight: test.blockHeight,
			}

			res, err := keeper.GetSession(ctx, req)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.expectedErrMsg)
			require.Equal(t, expectedRes, res)
		})
	}
}

package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/supplier"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	testServiceId = "svc1"
	testSessionId = "mock_session_id"
)

func TestMsgServer_CreateClaim_Success(t *testing.T) {
	appSupplierPair := supplier.AppSupplierPair{
		AppAddr:      sample.AccAddress(),
		SupplierAddr: sample.AccAddress(),
	}
	service := &sharedtypes.Service{Id: testServiceId}
	sessionFixturesByAddr := supplier.NewSessionFixturesWithPairings(t, service, appSupplierPair)

	supplierKeeper, sdkCtx := keepertest.SupplierKeeper(t, sessionFixturesByAddr)
	srv := keeper.NewMsgServerImpl(*supplierKeeper)
	ctx := sdk.WrapSDKContext(sdkCtx)

	claimMsg := newTestClaimMsg(t, testSessionId)
	claimMsg.SupplierAddress = appSupplierPair.SupplierAddr
	claimMsg.SessionHeader.ApplicationAddress = appSupplierPair.AppAddr

	createClaimRes, err := srv.CreateClaim(ctx, claimMsg)
	require.NoError(t, err)
	require.NotNil(t, createClaimRes)

	claimRes, err := supplierKeeper.AllClaims(sdkCtx, &types.QueryAllClaimsRequest{})
	require.NoError(t, err)

	claims := claimRes.GetClaim()
	require.Lenf(t, claims, 1, "expected 1 claim, got %d", len(claims))
	require.Equal(t, claimMsg.SessionHeader.SessionId, claims[0].GetSessionHeader().GetSessionId())
	require.Equal(t, claimMsg.SupplierAddress, claims[0].GetSupplierAddress())
	require.Equal(t, claimMsg.SessionHeader.GetSessionEndBlockHeight(), claims[0].GetSessionHeader().GetSessionEndBlockHeight())
	require.Equal(t, claimMsg.RootHash, claims[0].GetRootHash())
}

func TestMsgServer_CreateClaim_Error(t *testing.T) {
	service := &sharedtypes.Service{Id: testServiceId}
	appSupplierPair := supplier.AppSupplierPair{
		AppAddr:      sample.AccAddress(),
		SupplierAddr: sample.AccAddress(),
	}
	sessionFixturesByAppAddr := supplier.NewSessionFixturesWithPairings(t, service, appSupplierPair)

	supplierKeeper, sdkCtx := keepertest.SupplierKeeper(t, sessionFixturesByAppAddr)
	srv := keeper.NewMsgServerImpl(*supplierKeeper)
	ctx := sdk.WrapSDKContext(sdkCtx)

	tests := []struct {
		desc        string
		claimMsgFn  func(t *testing.T) *types.MsgCreateClaim
		expectedErr error
	}{
		{
			desc: "on-chain session ID must match claim msg session ID",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				msg := newTestClaimMsg(t, "invalid_session_id")
				msg.SupplierAddress = appSupplierPair.SupplierAddr
				msg.SessionHeader.ApplicationAddress = appSupplierPair.AppAddr

				return msg
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrSupplierInvalidSessionId.Wrapf(
					"session ID does not match on-chain session ID; expected %q, got %q",
					testSessionId,
					"invalid_session_id",
				).Error(),
			),
		},
		{
			desc: "claim msg supplier address must be in the session",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				msg := newTestClaimMsg(t, testSessionId)
				msg.SessionHeader.ApplicationAddress = appSupplierPair.AppAddr

				// Overwrite supplier address to one not included in the session fixtures.
				msg.SupplierAddress = sample.AccAddress()

				return msg
			},
			expectedErr: types.ErrSupplierNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			createClaimRes, err := srv.CreateClaim(ctx, tt.claimMsgFn(t))
			require.ErrorContains(t, err, tt.expectedErr.Error())
			require.Nil(t, createClaimRes)
		})
	}
}

func newTestClaimMsg(t *testing.T, sessionId string) *suppliertypes.MsgCreateClaim {
	t.Helper()

	return suppliertypes.NewMsgCreateClaim(
		sample.AccAddress(),
		&sessiontypes.SessionHeader{
			ApplicationAddress:      sample.AccAddress(),
			SessionStartBlockHeight: 0,
			SessionId:               sessionId,
			Service: &sharedtypes.Service{
				Id:   "svc1",
				Name: "svc1",
			},
		},
		[]byte{0, 0, 0, 0},
	)
}

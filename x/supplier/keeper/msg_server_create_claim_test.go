package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func TestMsgServer_CreateClaim_Success(t *testing.T) {
	appAddr, supplierAddr := sample.AccAddress(), sample.AccAddress()

	// TODO_IN_THIS_COMMIT: dedup & refactor to a test helper.
	sessionMockMap := keepertest.SessionMetaFixturesByAppAddr{
		appAddr: keepertest.SessionMetaFixture{
			SessionId:    "mock_session_id",
			AppAddr:      appAddr,
			SupplierAddr: supplierAddr,
		},
	}

	supplierKeeper, sdkCtx := keepertest.SupplierKeeper(t, sessionMockMap)
	srv := keeper.NewMsgServerImpl(*supplierKeeper)
	ctx := sdk.WrapSDKContext(sdkCtx)

	claimMsg := newTestClaimMsg(t)
	claimMsg.SupplierAddress = supplierAddr
	claimMsg.SessionHeader.ApplicationAddress = appAddr

	createClaimRes, err := srv.CreateClaim(ctx, claimMsg)
	require.NoError(t, err)
	require.NotNil(t, createClaimRes)

	claimRes, err := supplierKeeper.AllClaims(sdkCtx, &types.QueryAllClaimsRequest{})
	require.NoError(t, err)

	claims := claimRes.GetClaim()
	require.Lenf(t, claims, 1, "expected 1 claim, got %d", len(claims))
	require.Equal(t, claimMsg.SessionHeader.SessionId, claims[0].SessionId)
	require.Equal(t, claimMsg.SupplierAddress, claims[0].SupplierAddress)
	require.Equal(t, uint64(claimMsg.SessionHeader.GetSessionEndBlockHeight()), claims[0].SessionEndBlockHeight)
	require.Equal(t, claimMsg.RootHash, claims[0].RootHash)
}

func TestMsgServer_CreateClaim_Error(t *testing.T) {
	appAddr, supplierAddr := sample.AccAddress(), sample.AccAddress()

	// TODO_IN_THIS_COMMIT: dedup & refactor to a test helper.
	sessionMockMap := keepertest.SessionMetaFixturesByAppAddr{
		appAddr: keepertest.SessionMetaFixture{
			SessionId:    "mock_session_id",
			AppAddr:      appAddr,
			SupplierAddr: supplierAddr,
		},
	}
	supplierKeeper, sdkCtx := keepertest.SupplierKeeper(t, sessionMockMap)
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
				msg := newTestClaimMsg(t)
				msg.SupplierAddress = supplierAddr
				msg.SessionHeader.ApplicationAddress = appAddr
				msg.SessionHeader.SessionId = "invalid_session_id"

				return msg
			},
			expectedErr: types.ErrSupplierInvalidSessionId,
		},
		{
			desc: "claim msg supplier address must be in the session",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				msg := newTestClaimMsg(t)
				msg.SessionHeader.ApplicationAddress = appAddr

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
			require.ErrorIs(t, err, tt.expectedErr)
			require.Nil(t, createClaimRes)
		})
	}
}

func newTestClaimMsg(t *testing.T) *suppliertypes.MsgCreateClaim {
	t.Helper()

	return suppliertypes.NewMsgCreateClaim(
		sample.AccAddress(),
		&sessiontypes.SessionHeader{
			ApplicationAddress:      sample.AccAddress(),
			SessionStartBlockHeight: 1,
			SessionId:               "mock_session_id",
			Service: &sharedtypes.Service{
				Id:   "svc1",
				Name: "svc1",
			},
		},
		[]byte{0, 0, 0, 0},
	)
}

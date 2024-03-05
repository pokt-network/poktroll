package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	"github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestMsgServer_CreateClaim_Success(t *testing.T) {
	proofKeeperWithDeps, ctx := keepertest.NewProofKeeperWithDeps(t)
	proofKeeper := proofKeeperWithDeps.ProofKeeper
	srv := keeper.NewMsgServerImpl(*proofKeeper)

	service := &sharedtypes.Service{Id: testServiceId}
	supplierAddr := sample.AccAddress()
	appAddr := sample.AccAddress()

	supplierKeeper := proofKeeperWithDeps.SupplierKeeper
	appKeeper := proofKeeperWithDeps.ApplicationKeeper

	supplierKeeper.SetSupplier(ctx, sharedtypes.Supplier{
		Address: supplierAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{Service: service},
		},
	})

	appKeeper.SetApplication(ctx, apptypes.Application{
		Address: appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{Service: service},
		},
	})

	sessionRes, err := proofKeeperWithDeps.SessionKeeper.GetSession(
		ctx,
		&sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			Service:            service,
			BlockHeight:        1,
		},
	)
	require.NoError(t, err)

	claimMsg := newTestClaimMsg(t, sessionRes.GetSession().GetSessionId(), supplierAddr, appAddr, service)
	createClaimRes, err := srv.CreateClaim(ctx, claimMsg)
	require.NoError(t, err)
	require.NotNil(t, createClaimRes)

	claimRes, err := proofKeeper.AllClaims(ctx, &types.QueryAllClaimsRequest{})
	require.NoError(t, err)

	claims := claimRes.GetClaims()
	require.Lenf(t, claims, 1, "expected 1 claim, got %d", len(claims))
	require.Equal(t, claimMsg.SessionHeader.SessionId, claims[0].GetSessionHeader().GetSessionId())
	require.Equal(t, claimMsg.SupplierAddress, claims[0].GetSupplierAddress())
	require.Equal(t, claimMsg.SessionHeader.GetSessionEndBlockHeight(), claims[0].GetSessionHeader().GetSessionEndBlockHeight())
	require.Equal(t, claimMsg.RootHash, claims[0].GetRootHash())
}

func TestMsgServer_CreateClaim_Error(t *testing.T) {
	proofKeeperWithDeps, ctx := keepertest.NewProofKeeperWithDeps(t)
	proofKeeper := proofKeeperWithDeps.ProofKeeper
	srv := keeper.NewMsgServerImpl(*proofKeeper)

	service := &sharedtypes.Service{Id: "svc1"}
	supplierAddr, wrongSupplierAddr := sample.AccAddress(), sample.AccAddress()
	appAddr, _ := sample.AccAddress(), sample.AccAddress()
	supplierKeeper := proofKeeperWithDeps.SupplierKeeper

	// Add a supplier expected to be in the session.
	supplierKeeper.SetSupplier(ctx, sharedtypes.Supplier{
		Address: supplierAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{Service: service},
		},
	})

	// Add a supplier *not* expected to be in the session.
	supplierKeeper.SetSupplier(ctx, sharedtypes.Supplier{
		Address: wrongSupplierAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{Service: &sharedtypes.Service{Id: "nosvc1"}},
		},
	})

	proofKeeperWithDeps.ApplicationKeeper.SetApplication(ctx, apptypes.Application{
		Address: appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{Service: service},
		},
	})

	sessionRes, err := proofKeeperWithDeps.SessionKeeper.GetSession(
		ctx,
		&sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			Service:            service,
			BlockHeight:        1,
		},
	)
	require.NoError(t, err)

	tests := []struct {
		desc        string
		claimMsgFn  func(t *testing.T) *types.MsgCreateClaim
		expectedErr error
	}{
		{
			desc: "on-chain session ID must match claim msg session ID",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				return newTestClaimMsg(t,
					// Use a session ID that doesn't match.
					"invalid_session_id",
					supplierAddr,
					appAddr,
					service,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrProofInvalidSessionId.Wrapf(
					"session ID does not match on-chain session ID; expected %q, got %q",
					sessionRes.GetSession().GetSessionId(),
					"invalid_session_id",
				).Error(),
			),
		},
		{
			desc: "claim msg supplier address must be in the session",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				return newTestClaimMsg(t,
					sessionRes.GetSession().GetSessionId(),
					// Use a supplier address not included in the session.
					wrongSupplierAddr,
					appAddr,
					service,
				)
			},
			expectedErr: types.ErrProofNotFound,
		},
		{
			desc: "claim msg supplier address must exist on-chain",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				return newTestClaimMsg(t,
					sessionRes.GetSession().GetSessionId(),
					// Use a supplier address that's nonexistent on-chain.
					sample.AccAddress(),
					appAddr,
					service,
				)
			},
			expectedErr: types.ErrProofNotFound,
		},
		// TODO_IN_THIS_COMMIT: set correct expectations and uncomment.
		//{
		//	desc: "claim msg application address must be in the session",
		//	claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
		//		return newTestClaimMsg(t,
		//			sessionRes.GetSession().GetSessionId(),
		//			supplierAddr,
		//			// Use an application address not included in the session.
		//			wrongAppAddr,
		//			service,
		//		)
		//	},
		//	expectedErr: types.ErrProofNotFound,
		//},
		//{
		//	desc: "claim msg application address must exist on-chain",
		//	claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
		//		return newTestClaimMsg(t,
		//			sessionRes.GetSession().GetSessionId(),
		//			supplierAddr,
		//			// Use an application address that's nonexistent on-chain.
		//			sample.AccAddress(),
		//			service,
		//		)
		//	},
		//	expectedErr: types.ErrProofNotFound,
		//},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			createClaimRes, err := srv.CreateClaim(ctx, test.claimMsgFn(t))
			require.ErrorContains(t, err, test.expectedErr.Error())
			require.Nil(t, createClaimRes)
		})
	}
}

func newTestClaimMsg(
	t *testing.T,
	sessionId string,
	supplierAddr string,
	appAddr string,
	service *sharedtypes.Service,
) *types.MsgCreateClaim {
	t.Helper()

	return types.NewMsgCreateClaim(
		supplierAddr,
		&sessiontypes.SessionHeader{
			ApplicationAddress:      appAddr,
			SessionId:               sessionId,
			Service:                 service,
			SessionStartBlockHeight: 1,
		},
		[]byte{0, 0, 0, 0},
	)
}

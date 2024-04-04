package keeper_test

import (
	"testing"

	"github.com/pokt-network/smt"
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

var defaultMerkleRoot = []byte{0, 1, 0, 1}

func TestMsgServer_CreateClaim_Success(t *testing.T) {
	// Set block height to 1 so there is a valid session on-chain.
	blockHeightOpt := keepertest.WithBlockHeight(1)
	keepers, ctx := keepertest.NewProofModuleKeepers(t, blockHeightOpt)
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)

	service := &sharedtypes.Service{Id: testServiceId}
	supplierAddr := sample.AccAddress()
	appAddr := sample.AccAddress()

	keepers.SetSupplier(ctx, sharedtypes.Supplier{
		Address: supplierAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{Service: service},
		},
	})

	keepers.SetApplication(ctx, apptypes.Application{
		Address: appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{Service: service},
		},
	})

	sessionRes, err := keepers.GetSession(
		ctx,
		&sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			Service:            service,
			BlockHeight:        1,
		},
	)
	require.NoError(t, err)

	claimMsg := newTestClaimMsg(t,
		sessionRes.GetSession().GetSessionId(),
		supplierAddr,
		appAddr,
		service,
		defaultMerkleRoot,
	)
	createClaimRes, err := srv.CreateClaim(ctx, claimMsg)
	require.NoError(t, err)
	require.NotNil(t, createClaimRes)

	claimRes, err := keepers.AllClaims(ctx, &types.QueryAllClaimsRequest{})
	require.NoError(t, err)

	claims := claimRes.GetClaims()
	require.Lenf(t, claims, 1, "expected 1 claim, got %d", len(claims))
	require.Equal(t, claimMsg.SessionHeader.SessionId, claims[0].GetSessionHeader().GetSessionId())
	require.Equal(t, claimMsg.SupplierAddress, claims[0].GetSupplierAddress())
	require.Equal(t, claimMsg.SessionHeader.GetSessionEndBlockHeight(), claims[0].GetSessionHeader().GetSessionEndBlockHeight())
	require.Equal(t, claimMsg.RootHash, claims[0].GetRootHash())
}

func TestMsgServer_CreateClaim_Error(t *testing.T) {
	// Set block height to 1 so there is a valid session on-chain.
	blockHeightOpt := keepertest.WithBlockHeight(1)
	keepers, ctx := keepertest.NewProofModuleKeepers(t, blockHeightOpt)
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)

	// service is the only service for which a session should exist.
	service := &sharedtypes.Service{Id: testServiceId}
	// supplierAddr is staked for "svc1" such that it is expected to be in the session.
	supplierAddr := sample.AccAddress()
	// wrongSupplierAddr is staked for "nosvc1" such that it is *not* expected to be in the session.
	wrongSupplierAddr := sample.AccAddress()
	// randSupplierAddr is *not* staked for any service.
	randSupplierAddr := sample.AccAddress()

	// appAddr is staked for "svc1" such that it is expected to be in the session.
	appAddr := sample.AccAddress()
	// wrongAppAddr is staked for "nosvc1" such that it is *not* expected to be in the session.
	wrongAppAddr := sample.AccAddress()
	// randAppAddr is *not* staked for any service.
	randAppAddr := sample.AccAddress()

	supplierKeeper := keepers.SupplierKeeper
	appKeeper := keepers.ApplicationKeeper

	// Add a supplier that is expected to be in the session.
	supplierKeeper.SetSupplier(ctx, sharedtypes.Supplier{
		Address: supplierAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{Service: service},
		},
	})

	// Add a supplier that is *not* expected to be in the session.
	supplierKeeper.SetSupplier(ctx, sharedtypes.Supplier{
		Address: wrongSupplierAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{Service: &sharedtypes.Service{Id: "nosvc1"}},
		},
	})

	// Add an application that is expected to be in the session.
	appKeeper.SetApplication(ctx, apptypes.Application{
		Address: appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{Service: service},
		},
	})

	// Add an application that is *not* expected to be in the session.
	appKeeper.SetApplication(ctx, apptypes.Application{
		Address: wrongAppAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{Service: &sharedtypes.Service{Id: "nosvc1"}},
		},
	})

	// Query for the session which contains the expected app and supplier pair.
	sessionRes, err := keepers.SessionKeeper.GetSession(
		ctx,
		&sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			Service:            service,
			BlockHeight:        1,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, sessionRes)
	require.Equal(t, appAddr, sessionRes.GetSession().GetApplication().GetAddress())

	sessionResSuppliers := sessionRes.GetSession().GetSuppliers()
	require.NotEmpty(t, sessionResSuppliers)
	require.Equal(t, supplierAddr, sessionResSuppliers[0].GetAddress())

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
					defaultMerkleRoot,
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
					defaultMerkleRoot,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrProofNotFound.Wrapf(
					"supplier address %q not found in session ID %q",
					wrongSupplierAddr,
					sessionRes.GetSession().GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "claim msg supplier address must exist on-chain",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				return newTestClaimMsg(t,
					sessionRes.GetSession().GetSessionId(),
					// Use a supplier address that's nonexistent on-chain.
					randSupplierAddr,
					appAddr,
					service,
					defaultMerkleRoot,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrProofNotFound.Wrapf(
					"supplier address %q not found in session ID %q",
					randSupplierAddr,
					sessionRes.GetSession().GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "claim msg application address must be in the session",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				return newTestClaimMsg(t,
					sessionRes.GetSession().GetSessionId(),
					supplierAddr,
					// Use an application address not included in the session.
					wrongAppAddr,
					service,
					defaultMerkleRoot,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				sessiontypes.ErrSessionAppNotStakedForService.Wrapf(
					"application %q not staked for service ID %q",
					wrongAppAddr,
					service.GetId(),
				).Error(),
			),
		},
		{
			desc: "claim msg application address must exist on-chain",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				return newTestClaimMsg(t,
					sessionRes.GetSession().GetSessionId(),
					supplierAddr,
					// Use an application address that's nonexistent on-chain.
					randAppAddr,
					service,
					defaultMerkleRoot,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				sessiontypes.ErrSessionAppNotFound.Wrapf(
					"could not find app with address %q at height %d",
					randAppAddr,
					sessionRes.GetSession().GetHeader().GetSessionStartBlockHeight(),
				).Error(),
			),
		},
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
	merkleRoot smt.MerkleRoot,
) *types.MsgCreateClaim {
	t.Helper()

	return types.NewMsgCreateClaim(
		supplierAddr,
		&sessiontypes.SessionHeader{
			ApplicationAddress:      appAddr,
			SessionId:               sessionId,
			Service:                 service,
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   4,
		},
		merkleRoot,
	)
}

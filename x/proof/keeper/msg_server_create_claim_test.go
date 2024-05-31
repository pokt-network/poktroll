package keeper_test

import (
	"context"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	"github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var defaultMerkleRoot = []byte{0, 1, 0, 1}

func TestMsgServer_CreateClaim_Success(t *testing.T) {
	tests := []struct {
		desc           string
		getClaimHeight func(
			ctx context.Context,
			keepers *keepertest.ProofModuleKeepers,
			sessionHeader *sessiontypes.SessionHeader,
		) int64
	}{
		{
			desc: "claim window open height",
			getClaimHeight: func(
				ctx context.Context,
				keepers *keepertest.ProofModuleKeepers,
				sessionHeader *sessiontypes.SessionHeader,
			) int64 {
				sharedParams := keepers.SharedKeeper.GetParams(ctx)
				return shared.GetClaimWindowOpenHeight(
					&sharedParams,
					sessionHeader.GetSessionEndBlockHeight(),
				)
			},
		},
		{
			desc: "claim window close height minus one",
			getClaimHeight: func(
				ctx context.Context,
				keepers *keepertest.ProofModuleKeepers,
				sessionHeader *sessiontypes.SessionHeader,
			) int64 {
				sharedParams := keepers.SharedKeeper.GetParams(ctx)
				return shared.GetClaimWindowCloseHeight(
					&sharedParams,
					sessionHeader.GetSessionEndBlockHeight(),
				)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Set block height to 1 so there is a valid session on-chain.
			blockHeightOpt := keepertest.WithBlockHeight(1)
			keepers, ctx := keepertest.NewProofModuleKeepers(t, blockHeightOpt)
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			srv := keeper.NewMsgServerImpl(*keepers.Keeper)

			// The base session start height used for testing
			sessionStartHeight := int64(1)

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

			sessionHeader := sessionRes.GetSession().GetHeader()

			// Increment the block height to the claim window open height.
			sdkCtx = sdkCtx.WithBlockHeight(test.getClaimHeight(ctx, keepers, sessionHeader))
			ctx = sdkCtx

			claimMsg := newTestClaimMsg(t,
				sessionStartHeight,
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
		})
	}
}

func TestMsgServer_CreateClaim_OutsideOfWindow(t *testing.T) {
	// Set block height to 1 so there is a valid session on-chain.
	blockHeightOpt := keepertest.WithBlockHeight(1)
	keepers, ctx := keepertest.NewProofModuleKeepers(t, blockHeightOpt)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)
	sharedParams := keepers.SharedKeeper.GetParams(ctx)

	// The base session start height used for testing
	sessionStartHeight := int64(1)

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

	sessionHeader := sessionRes.GetSession().GetHeader()

	claimWindowOpenHeight := shared.GetClaimWindowOpenHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
	)

	// Increment the block height to one block before the claim window open height.
	sdkCtx = sdkCtx.WithBlockHeight(claimWindowOpenHeight - 1)
	ctx = sdkCtx

	// Attempt to create a claim before the claim window open height.
	claimMsg := newTestClaimMsg(t,
		sessionStartHeight,
		sessionRes.GetSession().GetSessionId(),
		supplierAddr,
		appAddr,
		service,
		defaultMerkleRoot,
	)
	_, err = srv.CreateClaim(ctx, claimMsg)
	require.ErrorContains(t, err, types.ErrProofClaimOutsideOfWindow.Wrapf(
		"current block height %d is less than session claim window open height %d",
		sdkCtx.BlockHeight(),
		shared.GetClaimWindowOpenHeight(&sharedParams, sessionHeader.GetSessionEndBlockHeight()),
	).Error())

	claimRes, err := keepers.AllClaims(ctx, &types.QueryAllClaimsRequest{})
	require.NoError(t, err)

	claims := claimRes.GetClaims()
	require.Lenf(t, claims, 0, "expected 0 claim, got %d", len(claims))

	claimWindowCloseHeight := shared.GetClaimWindowCloseHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
	)

	// Increment the block height to one block after the claim window close height.
	sdkCtx = sdkCtx.WithBlockHeight(claimWindowCloseHeight + 1)
	ctx = sdkCtx

	// Attempt to create a claim after the claim window close height.
	claimMsg = newTestClaimMsg(t,
		sessionStartHeight,
		sessionRes.GetSession().GetSessionId(),
		supplierAddr,
		appAddr,
		service,
		defaultMerkleRoot,
	)
	_, err = srv.CreateClaim(ctx, claimMsg)
	require.ErrorContains(t, err, types.ErrProofClaimOutsideOfWindow.Wrapf(
		"current block height %d is greater than session claim window close height %d",
		sdkCtx.BlockHeight(),
		claimWindowCloseHeight,
	).Error())

	claimRes, err = keepers.AllClaims(ctx, &types.QueryAllClaimsRequest{})
	require.NoError(t, err)

	claims = claimRes.GetClaims()
	require.Lenf(t, claims, 0, "expected 0 claim, got %d", len(claims))
}

func TestMsgServer_CreateClaim_Error(t *testing.T) {
	// Set block height to 1 so there is a valid session on-chain.
	blockHeightOpt := keepertest.WithBlockHeight(1)
	keepers, ctx := keepertest.NewProofModuleKeepers(t, blockHeightOpt)
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)

	// The base session start height used for testing
	sessionStartHeight := int64(1)
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
					sessionStartHeight,
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
					sessionStartHeight,
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
					sessionStartHeight,
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
					sessionStartHeight,
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
					sessionStartHeight,
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
	sessionStartHeight int64,
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
			Service:                 service,
			SessionId:               sessionId,
			SessionStartBlockHeight: sessionStartHeight,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(sessionStartHeight),
		},
		merkleRoot,
	)
}

package keeper_test

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	"github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	expectedNumRelays       = 10
	computeUnitsPerRelay    = 1
	expectedNumComputeUnits = expectedNumRelays * computeUnitsPerRelay
)

var defaultMerkleRoot = testproof.SmstRootWithSumAndCount(expectedNumComputeUnits, expectedNumRelays)

func TestMsgServer_CreateClaim_Success(t *testing.T) {
	var claimWindowOpenBlockHash []byte
	supplierAddr := sample.AccAddress()

	tests := []struct {
		desc              string
		getClaimMsgHeight func(
			sharedParams *sharedtypes.Params,
			queryHeight int64,
		) int64
	}{
		{
			desc: "claim message height equals supplier's earliest claim commit height",
			getClaimMsgHeight: func(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
				return shared.GetEarliestSupplierClaimCommitHeight(
					sharedParams,
					queryHeight,
					claimWindowOpenBlockHash,
					supplierAddr,
				)
			},
		},
		{
			desc:              "claim message height equals claim window close height",
			getClaimMsgHeight: shared.GetClaimWindowCloseHeight,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Set block height to 1 so there is a valid session on-chain.
			blockHeight := int64(1)
			blockHeightOpt := keepertest.WithBlockHeight(blockHeight)

			// Create a new set of proof module keepers. This isolates each test
			// case from side effects of other test cases.
			keepers, ctx := keepertest.NewProofModuleKeepers(t, blockHeightOpt)
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			srv := keeper.NewMsgServerImpl(*keepers.Keeper)

			// The base session start height used for testing
			sessionStartHeight := blockHeight

			service := &sharedtypes.Service{
				Id:                   testServiceId,
				ComputeUnitsPerRelay: computeUnitsPerRelay,
				OwnerAddress:         sample.AccAddress(),
			}
			appAddr := sample.AccAddress()

			keepers.SetSupplier(ctx, sharedtypes.Supplier{
				OperatorAddress: supplierAddr,
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
					BlockHeight:        blockHeight,
				},
			)
			require.NoError(t, err)

			sessionHeader := sessionRes.GetSession().GetHeader()

			// Increment the block height to the test claim height.
			sharedParams := keepers.SharedKeeper.GetParams(ctx)
			testClaimHeight := test.getClaimMsgHeight(&sharedParams, sessionHeader.GetSessionEndBlockHeight())
			sdkCtx = sdkCtx.WithBlockHeight(testClaimHeight)
			ctx = sdkCtx

			// Create a claim.
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

			// Query for all claims.
			claimRes, err := keepers.AllClaims(ctx, &types.QueryAllClaimsRequest{})
			require.NoError(t, err)

			claims := claimRes.GetClaims()
			require.Lenf(t, claims, 1, "expected 1 claim, got %d", len(claims))

			// Ensure that the claim was created successfully and assert that it
			// matches the MsgCreateClaim.
			claim := claims[0]
			claimSessionHeader := claim.GetSessionHeader()
			require.Equal(t, claimMsg.SessionHeader.SessionId, claimSessionHeader.GetSessionId())
			require.Equal(t, claimMsg.SupplierOperatorAddress, claim.GetSupplierOperatorAddress())
			require.Equal(t, claimMsg.SessionHeader.GetSessionEndBlockHeight(), claimSessionHeader.GetSessionEndBlockHeight())
			require.Equal(t, claimMsg.RootHash, claim.GetRootHash())

			events := sdkCtx.EventManager().Events()
			require.Equal(t, 1, len(events))

			require.Equal(t, "poktroll.proof.EventClaimCreated", events[0].Type)

			event, err := cosmostypes.ParseTypedEvent(abci.Event(events[0]))
			require.NoError(t, err)

			claimCreatedEvent, ok := event.(*types.EventClaimCreated)
			require.Truef(t, ok, "unexpected event type %T", event)

			require.EqualValues(t, &claim, claimCreatedEvent.GetClaim())
			require.Equal(t, uint64(expectedNumComputeUnits), claimCreatedEvent.GetNumComputeUnits())
			require.Equal(t, uint64(expectedNumRelays), claimCreatedEvent.GetNumRelays())
		})
	}
}

func TestMsgServer_CreateClaim_Error_OutsideOfWindow(t *testing.T) {
	var claimWindowOpenBlockHash []byte

	// Set block height to 1 so there is a valid session on-chain.
	blockHeightOpt := keepertest.WithBlockHeight(1)
	keepers, ctx := keepertest.NewProofModuleKeepers(t, blockHeightOpt)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)
	sharedParams := keepers.SharedKeeper.GetParams(ctx)

	// The base session start height used for testing
	sessionStartHeight := int64(1)

	service := &sharedtypes.Service{
		Id:                   testServiceId,
		ComputeUnitsPerRelay: computeUnitsPerRelay,
		OwnerAddress:         sample.AccAddress(),
	}
	supplierAddr := sample.AccAddress()
	appAddr := sample.AccAddress()

	keepers.SetSupplier(ctx, sharedtypes.Supplier{
		OperatorAddress: supplierAddr,
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

	claimWindowCloseHeight := shared.GetClaimWindowCloseHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
	)

	earliestClaimCommitHeight := shared.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
		claimWindowOpenBlockHash,
		supplierAddr,
	)

	tests := []struct {
		desc           string
		claimMsgHeight int64
		expectedErr    error
	}{
		{
			desc:           "claim message height equals claim window open height minus one",
			claimMsgHeight: earliestClaimCommitHeight - 1,
			expectedErr: status.Error(
				codes.FailedPrecondition,
				types.ErrProofClaimOutsideOfWindow.Wrapf(
					"current block height (%d) is less than the session's earliest claim commit height (%d)",
					earliestClaimCommitHeight-1,
					shared.GetEarliestSupplierClaimCommitHeight(
						&sharedParams,
						sessionHeader.GetSessionEndBlockHeight(),
						claimWindowOpenBlockHash,
						supplierAddr,
					),
				).Error(),
			),
		},
		{
			desc:           "claim message height equals claim window close height plus one",
			claimMsgHeight: claimWindowCloseHeight + 1,
			expectedErr: status.Error(
				codes.FailedPrecondition,
				types.ErrProofClaimOutsideOfWindow.Wrapf(
					"current block height (%d) is greater than session claim window close height (%d)",
					claimWindowCloseHeight+1,
					claimWindowCloseHeight,
				).Error(),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// Increment the block height to the test claim height.
			sdkCtx = sdkCtx.WithBlockHeight(test.claimMsgHeight)
			ctx = sdkCtx

			// Attempt to create a claim at the test claim height.
			claimMsg := newTestClaimMsg(t,
				sessionStartHeight,
				sessionRes.GetSession().GetSessionId(),
				supplierAddr,
				appAddr,
				service,
				defaultMerkleRoot,
			)
			_, err = srv.CreateClaim(ctx, claimMsg)
			require.ErrorContains(t, err, test.expectedErr.Error())

			claimRes, err := keepers.AllClaims(ctx, &types.QueryAllClaimsRequest{})
			require.NoError(t, err)

			claims := claimRes.GetClaims()
			require.Lenf(t, claims, 0, "expected 0 claim, got %d", len(claims))

			// Assert that no events were emitted.
			events := sdkCtx.EventManager().Events()
			require.Equal(t, 0, len(events))
		})
	}
}

func TestMsgServer_CreateClaim_Error(t *testing.T) {
	// Set block height to 1 so there is a valid session on-chain.
	blockHeightOpt := keepertest.WithBlockHeight(1)
	keepers, ctx := keepertest.NewProofModuleKeepers(t, blockHeightOpt)
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)

	// The base session start height used for testing
	sessionStartHeight := int64(1)
	// service is the only service for which a session should exist.
	service := &sharedtypes.Service{
		Id:                   testServiceId,
		ComputeUnitsPerRelay: computeUnitsPerRelay,
		OwnerAddress:         sample.AccAddress(),
	}
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
		OperatorAddress: supplierAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{Service: service},
		},
	})

	// Add a supplier that is *not* expected to be in the session.
	supplierKeeper.SetSupplier(ctx, sharedtypes.Supplier{
		OperatorAddress: wrongSupplierAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				Service: &sharedtypes.Service{
					Id:                   "nosvc1",
					ComputeUnitsPerRelay: computeUnitsPerRelay,
					OwnerAddress:         sample.AccAddress(),
				},
			},
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
			{
				Service: &sharedtypes.Service{
					Id:                   "nosvc1",
					ComputeUnitsPerRelay: computeUnitsPerRelay,
					OwnerAddress:         sample.AccAddress(),
				},
			},
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
	require.Equal(t, supplierAddr, sessionResSuppliers[0].GetOperatorAddress())

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
					"supplier operator address %q not found in session ID %q",
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
					"supplier operator address %q not found in session ID %q",
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

			// Assert that no events were emitted.
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			events := sdkCtx.EventManager().Events()
			require.Equal(t, 0, len(events))
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
	merkleRoot smt.MerkleSumRoot,
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

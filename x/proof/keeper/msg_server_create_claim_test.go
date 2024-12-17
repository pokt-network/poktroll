package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	"github.com/pokt-network/poktroll/x/proof/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	expectedNumRelays       = 10
	computeUnitsPerRelay    = 1
	expectedNumComputeUnits = expectedNumRelays * computeUnitsPerRelay

	nonDefaultComputeUnitsPerRelay    = 9999
	expectedNonDefaultNumComputeUnits = expectedNumRelays * nonDefaultComputeUnitsPerRelay
)

var (
	defaultMerkleRoot = testproof.SmstRootWithSumAndCount(expectedNumComputeUnits, expectedNumRelays)

	// Merkle root for Smst of a claim for the service with non-default compute units per relay
	customComputeUnitsPerRelayMerkleRoot = testproof.SmstRootWithSumAndCount(expectedNonDefaultNumComputeUnits, expectedNumRelays)
)

func TestMsgServer_CreateClaim_Success(t *testing.T) {
	var claimWindowOpenBlockHash []byte
	supplierOperatorAddr := sample.AccAddress()

	tests := []struct {
		desc              string
		getClaimMsgHeight func(
			sharedParams *sharedtypes.Params,
			queryHeight int64,
		) int64
		merkleRoot smt.MerkleSumRoot
		// The Compute Units Per Relay for the service used in the test.
		serviceComputeUnitsPerRelay    uint64
		expectedNumClaimedComputeUnits uint64
	}{
		{
			desc: "claim message height equals supplier's earliest claim commit height",
			getClaimMsgHeight: func(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
				return sharedtypes.GetEarliestSupplierClaimCommitHeight(
					sharedParams,
					queryHeight,
					claimWindowOpenBlockHash,
					supplierOperatorAddr,
				)
			},
			merkleRoot:                     defaultMerkleRoot,
			serviceComputeUnitsPerRelay:    computeUnitsPerRelay,
			expectedNumClaimedComputeUnits: expectedNumComputeUnits,
		},
		{
			desc:                           "claim message height equals claim window close height",
			getClaimMsgHeight:              sharedtypes.GetClaimWindowCloseHeight,
			merkleRoot:                     defaultMerkleRoot,
			serviceComputeUnitsPerRelay:    computeUnitsPerRelay,
			expectedNumClaimedComputeUnits: expectedNumComputeUnits,
		},
		{
			desc:                           "claim message for service with >1 compute units per relay",
			getClaimMsgHeight:              sharedtypes.GetClaimWindowCloseHeight,
			merkleRoot:                     customComputeUnitsPerRelayMerkleRoot,
			serviceComputeUnitsPerRelay:    nonDefaultComputeUnitsPerRelay,
			expectedNumClaimedComputeUnits: expectedNonDefaultNumComputeUnits,
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
				ComputeUnitsPerRelay: test.serviceComputeUnitsPerRelay,
				OwnerAddress:         sample.AccAddress(),
			}
			appAddr := sample.AccAddress()

			keepers.SetSupplier(ctx, sharedtypes.Supplier{
				OperatorAddress: supplierOperatorAddr,
				Services: []*sharedtypes.SupplierServiceConfig{
					{ServiceId: service.Id},
				},
			})

			keepers.SetApplication(ctx, apptypes.Application{
				Address: appAddr,
				ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
					{ServiceId: service.Id},
				},
			})

			keepers.SetService(ctx, *service)

			sessionRes, err := keepers.GetSession(
				ctx,
				&sessiontypes.QueryGetSessionRequest{
					ApplicationAddress: appAddr,
					ServiceId:          service.Id,
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
				supplierOperatorAddr,
				appAddr,
				service,
				test.merkleRoot,
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

			claimCreatedEvents := testutilevents.FilterEvents[*prooftypes.EventClaimCreated](t, events)
			require.Len(t, claimCreatedEvents, 1)

			targetNumRelays := keepers.ServiceKeeper.GetParams(ctx).TargetNumRelays
			relayMiningDifficulty := servicekeeper.NewDefaultRelayMiningDifficulty(
				ctx,
				keepers.Logger(),
				service.Id,
				targetNumRelays,
				targetNumRelays,
			)

			numEstimatedComputUnits, err := claim.GetNumEstimatedComputeUnits(relayMiningDifficulty)
			require.NoError(t, err)

			claimedUPOKT, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
			require.NoError(t, err)

			require.EqualValues(t, &claim, claimCreatedEvents[0].GetClaim())
			require.Equal(t, uint64(test.expectedNumClaimedComputeUnits), claimCreatedEvents[0].GetNumClaimedComputeUnits())
			require.Equal(t, uint64(expectedNumRelays), claimCreatedEvents[0].GetNumRelays())
			require.Equal(t, numEstimatedComputUnits, claimCreatedEvents[0].GetNumClaimedComputeUnits())
			require.Equal(t, &claimedUPOKT, claimCreatedEvents[0].GetClaimedUpokt())
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
	supplierOperatorAddr := sample.AccAddress()
	appAddr := sample.AccAddress()

	keepers.SetSupplier(ctx, sharedtypes.Supplier{
		OperatorAddress: supplierOperatorAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{ServiceId: service.Id},
		},
	})

	keepers.SetApplication(ctx, apptypes.Application{
		Address: appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: service.Id},
		},
	})

	sessionRes, err := keepers.GetSession(
		ctx,
		&sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			ServiceId:          service.Id,
			BlockHeight:        1,
		},
	)
	require.NoError(t, err)

	sessionHeader := sessionRes.GetSession().GetHeader()

	claimWindowCloseHeight := sharedtypes.GetClaimWindowCloseHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
	)

	earliestClaimCommitHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
		claimWindowOpenBlockHash,
		supplierOperatorAddr,
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
					sharedtypes.GetEarliestSupplierClaimCommitHeight(
						&sharedParams,
						sessionHeader.GetSessionEndBlockHeight(),
						claimWindowOpenBlockHash,
						supplierOperatorAddr,
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
				supplierOperatorAddr,
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
			claimCreatedEvents := testutilevents.FilterEvents[*prooftypes.EventClaimCreated](t, events)
			require.Len(t, claimCreatedEvents, 0)
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
	// supplierOperatorAddr is staked for "svc1" such that it is expected to be in the session.
	supplierOperatorAddr := sample.AccAddress()
	// wrongSupplierOperatorAddr is staked for "nosvc1" such that it is *not* expected to be in the session.
	wrongSupplierOperatorAddr := sample.AccAddress()
	// randSupplierOperatorAddr is *not* staked for any service.
	randSupplierOperatorAddr := sample.AccAddress()

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
		OperatorAddress: supplierOperatorAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{ServiceId: service.Id},
		},
	})

	// Add a supplier that is *not* expected to be in the session.
	supplierKeeper.SetSupplier(ctx, sharedtypes.Supplier{
		OperatorAddress: wrongSupplierOperatorAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{ServiceId: "nosvc1"},
		},
	})

	// Add an application that is expected to be in the session.
	appKeeper.SetApplication(ctx, apptypes.Application{
		Address: appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: service.Id},
		},
	})

	// Add an application that is *not* expected to be in the session.
	appKeeper.SetApplication(ctx, apptypes.Application{
		Address: wrongAppAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: "nosvc1"},
		},
	})

	// Query for the session which contains the expected app and supplier pair.
	sessionRes, err := keepers.SessionKeeper.GetSession(
		ctx,
		&sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			ServiceId:          service.Id,
			BlockHeight:        1,
		},
	)
	require.NoError(t, err)
	require.NoError(t, err)
	require.NotNil(t, sessionRes)
	require.Equal(t, appAddr, sessionRes.GetSession().GetApplication().GetAddress())

	sessionResSuppliers := sessionRes.GetSession().GetSuppliers()
	require.NotEmpty(t, sessionResSuppliers)
	require.Equal(t, supplierOperatorAddr, sessionResSuppliers[0].GetOperatorAddress())

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
					supplierOperatorAddr,
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
			desc: "claim msg supplier operator address must be in the session",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				return newTestClaimMsg(t,
					sessionStartHeight,
					sessionRes.GetSession().GetSessionId(),
					// Use a supplier operator address not included in the session.
					wrongSupplierOperatorAddr,
					appAddr,
					service,
					defaultMerkleRoot,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrProofNotFound.Wrapf(
					"supplier operator address %q not found in session ID %q",
					wrongSupplierOperatorAddr,
					sessionRes.GetSession().GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "claim msg supplier operator address must exist on-chain",
			claimMsgFn: func(t *testing.T) *types.MsgCreateClaim {
				return newTestClaimMsg(t,
					sessionStartHeight,
					sessionRes.GetSession().GetSessionId(),
					// Use a supplier operat address that's nonexistent on-chain.
					randSupplierOperatorAddr,
					appAddr,
					service,
					defaultMerkleRoot,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrProofNotFound.Wrapf(
					"supplier operator address %q not found in session ID %q",
					randSupplierOperatorAddr,
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
					supplierOperatorAddr,
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
					supplierOperatorAddr,
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
			claimCreatedEvents := testutilevents.FilterEvents[*prooftypes.EventClaimCreated](t, events)
			require.Len(t, claimCreatedEvents, 0)
		})
	}
}

func TestMsgServer_CreateClaim_Error_ComputeUnitsMismatch(t *testing.T) {
	// Set block height to 1 so there is a valid session on-chain.
	blockHeightOpt := keepertest.WithBlockHeight(1)
	keepers, ctx := keepertest.NewProofModuleKeepers(t, blockHeightOpt)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)

	// The base session start height used for testing
	sessionStartHeight := int64(1)

	// service is the only service for which a session should exist.
	// this service has a value of greater than 1 for the compute units per relay.
	service := &sharedtypes.Service{
		Id:                   testServiceId,
		ComputeUnitsPerRelay: nonDefaultComputeUnitsPerRelay,
		OwnerAddress:         sample.AccAddress(),
	}
	// Add the service that is expected to be on-chain.
	keepers.SetService(ctx, *service)

	// Add a supplier that is expected to be in the session.
	// supplierAddr is staked for "svc1" such that it is expected to be in the session.
	supplierKeeper := keepers.SupplierKeeper
	supplierAddr := sample.AccAddress()
	supplierKeeper.SetSupplier(ctx, sharedtypes.Supplier{
		OperatorAddress: supplierAddr,
		Services: []*sharedtypes.SupplierServiceConfig{
			{ServiceId: service.Id},
		},
	})

	// Add an application that is expected to be in the session.
	// appAddr is staked for "svc1" such that it is expected to be in the session.
	appKeeper := keepers.ApplicationKeeper
	appAddr := sample.AccAddress()
	appKeeper.SetApplication(ctx, apptypes.Application{
		Address: appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: service.Id},
		},
	})

	// Query for the session which contains the expected app and supplier pair.
	sessionRes, err := keepers.SessionKeeper.GetSession(
		ctx,
		&sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			ServiceId:          service.Id,
			BlockHeight:        1,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, sessionRes)
	require.Equal(t, appAddr, sessionRes.GetSession().GetApplication().GetAddress())

	sessionResSuppliers := sessionRes.GetSession().GetSuppliers()
	require.NotEmpty(t, sessionResSuppliers)
	require.Equal(t, supplierAddr, sessionResSuppliers[0].GetOperatorAddress())

	// Increment the block height to the test claim height.
	sessionHeader := sessionRes.GetSession().GetHeader()
	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	testClaimHeight := sharedtypes.GetClaimWindowCloseHeight(&sharedParams, sessionHeader.GetSessionEndBlockHeight())
	sdkCtx = sdkCtx.WithBlockHeight(testClaimHeight)
	ctx = sdkCtx

	// Prepare a message to submit a claim while also creating a new test claim
	testClaimMsg := newTestClaimMsg(t,
		sessionStartHeight,
		sessionRes.GetSession().GetSessionId(),
		supplierAddr,
		appAddr,
		service,
		defaultMerkleRoot,
	)

	// use the test claim message to create a claim object to get the number of relays and compute units in the claim.
	testClaim := types.Claim{RootHash: testClaimMsg.GetRootHash()}
	testClaimNumComputeUnits, err := testClaim.GetNumClaimedComputeUnits()
	require.NoError(t, err)
	testClaimNumRelays, err := testClaim.GetNumRelays()
	require.NoError(t, err)

	// Ensure that submitting the claim fails because the number of compute units
	// claimed does not match the expected amount as a function of (relay, service_CUPR)
	createClaimRes, err := srv.CreateClaim(ctx, testClaimMsg)
	require.ErrorContains(t,
		err,
		status.Error(
			codes.InvalidArgument,
			types.ErrProofComputeUnitsMismatch.Wrapf(
				"claim compute units: %d is not equal to number of relays %d * compute units per relay %d for service %s",
				testClaimNumComputeUnits,
				testClaimNumRelays,
				nonDefaultComputeUnitsPerRelay,
				sessionHeader.ServiceId,
			).Error(),
		).Error(),
	)

	require.Nil(t, createClaimRes)

	// Assert that no events were emitted.
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx)
	events := sdkCtx.EventManager().Events()
	claimCreatedEvents := testutilevents.FilterEvents[*prooftypes.EventClaimCreated](t, events)
	require.Len(t, claimCreatedEvents, 0)
}

func newTestClaimMsg(
	t *testing.T,
	sessionStartHeight int64,
	sessionId string,
	supplierOperatorAddr string,
	appAddr string,
	service *sharedtypes.Service,
	merkleRoot smt.MerkleSumRoot,
) *types.MsgCreateClaim {
	t.Helper()

	return types.NewMsgCreateClaim(
		supplierOperatorAddr,
		&sessiontypes.SessionHeader{
			ApplicationAddress:      appAddr,
			ServiceId:               service.Id,
			SessionId:               sessionId,
			SessionStartBlockHeight: sessionStartHeight,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(sessionStartHeight),
		},
		merkleRoot,
	)
}

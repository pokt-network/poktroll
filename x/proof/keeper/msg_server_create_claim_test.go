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
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/proof/keeper"
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
	supplierOperatorAddr := sample.AccAddressBech32()

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
			// Set block height to 1 so there is a valid session onchain.
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
				OwnerAddress:         sample.AccAddressBech32(),
			}
			appAddr := sample.AccAddressBech32()

			supplierServices := []*sharedtypes.SupplierServiceConfig{
				{ServiceId: service.Id},
			}
			serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(supplierOperatorAddr, supplierServices, 1, 0)
			keepers.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{
				OperatorAddress:      supplierOperatorAddr,
				Services:             supplierServices,
				ServiceConfigHistory: serviceConfigHistory,
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
			_, err = srv.CreateClaim(ctx, claimMsg)
			require.NoError(t, err)

			// Query for all claims.
			claimRes, err := keepers.AllClaims(ctx, &prooftypes.QueryAllClaimsRequest{})
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

			require.Equal(t, claim.SessionHeader.ServiceId, claimCreatedEvents[0].GetServiceId())
			require.Equal(t, claim.SessionHeader.ApplicationAddress, claimCreatedEvents[0].GetApplicationAddress())
			require.Equal(t, claim.SessionHeader.SessionEndBlockHeight, claimCreatedEvents[0].GetSessionEndBlockHeight())
			require.Equal(t, int32(prooftypes.ClaimProofStatus_PENDING_VALIDATION), claimCreatedEvents[0].GetClaimProofStatusInt())
			require.Equal(t, uint64(test.expectedNumClaimedComputeUnits), claimCreatedEvents[0].GetNumClaimedComputeUnits())
			require.Equal(t, uint64(expectedNumRelays), claimCreatedEvents[0].GetNumRelays())
			require.Equal(t, numEstimatedComputUnits, claimCreatedEvents[0].GetNumClaimedComputeUnits())
			require.Equal(t, claimedUPOKT.String(), claimCreatedEvents[0].GetClaimedUpokt())
		})
	}
}

func TestMsgServer_CreateClaim_Error_OutsideOfWindow(t *testing.T) {
	var claimWindowOpenBlockHash []byte

	// Set block height to 1 so there is a valid session onchain.
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
		OwnerAddress:         sample.AccAddressBech32(),
	}
	supplierOperatorAddr := sample.AccAddressBech32()
	appAddr := sample.AccAddressBech32()

	supplierServices := []*sharedtypes.SupplierServiceConfig{
		{ServiceId: service.Id},
	}
	supplierServiceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(supplierOperatorAddr, supplierServices, 1, 0)
	keepers.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{
		OperatorAddress:      supplierOperatorAddr,
		Services:             supplierServices,
		ServiceConfigHistory: supplierServiceConfigHistory,
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
				prooftypes.ErrProofClaimOutsideOfWindow.Wrapf(
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
				prooftypes.ErrProofClaimOutsideOfWindow.Wrapf(
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

			claimRes, err := keepers.AllClaims(ctx, &prooftypes.QueryAllClaimsRequest{})
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
	// Set block height to 1 so there is a valid session onchain.
	blockHeightOpt := keepertest.WithBlockHeight(1)
	keepers, ctx := keepertest.NewProofModuleKeepers(t, blockHeightOpt)
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)

	invalidRootHashWithZeroCount := testproof.RandSmstRootWithSumAndCount(t, 1, 0)
	invalidRootHashWithZeroSum := testproof.RandSmstRootWithSumAndCount(t, 0, 1)

	// The base session start height used for testing
	sessionStartHeight := int64(1)
	// service is the only service for which a session should exist.
	service := &sharedtypes.Service{
		Id:                   testServiceId,
		ComputeUnitsPerRelay: computeUnitsPerRelay,
		OwnerAddress:         sample.AccAddressBech32(),
	}
	// supplierOperatorAddr is staked for "svc1" such that it is expected to be in the session.
	supplierOperatorAddr := sample.AccAddressBech32()
	// wrongSupplierOperatorAddr is staked for "nosvc1" such that it is *not* expected to be in the session.
	wrongSupplierOperatorAddr := sample.AccAddressBech32()
	// randSupplierOperatorAddr is *not* staked for any service.
	randSupplierOperatorAddr := sample.AccAddressBech32()

	// appAddr is staked for "svc1" such that it is expected to be in the session.
	appAddr := sample.AccAddressBech32()
	// wrongAppAddr is staked for "nosvc1" such that it is *not* expected to be in the session.
	wrongAppAddr := sample.AccAddressBech32()
	// randAppAddr is *not* staked for any service.
	randAppAddr := sample.AccAddressBech32()

	supplierKeeper := keepers.SupplierKeeper
	appKeeper := keepers.ApplicationKeeper

	// Add a supplier that is expected to be in the session.
	supplierServices := []*sharedtypes.SupplierServiceConfig{
		{ServiceId: service.Id},
	}
	supplierServiceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(supplierOperatorAddr, supplierServices, 1, 0)
	supplierKeeper.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{
		OperatorAddress:      supplierOperatorAddr,
		Services:             supplierServices,
		ServiceConfigHistory: supplierServiceConfigHistory,
	})

	// Add a supplier that is *not* expected to be in the session.
	otherServices := []*sharedtypes.SupplierServiceConfig{
		{ServiceId: "nosvc1"},
	}
	wrongSupplierServiceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(wrongSupplierOperatorAddr, otherServices, 1, 0)
	supplierKeeper.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{
		OperatorAddress:      wrongSupplierOperatorAddr,
		Services:             otherServices,
		ServiceConfigHistory: wrongSupplierServiceConfigHistory,
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
	sessionRes, err := keepers.GetSession(
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
		claimMsgFn  func(t *testing.T) *prooftypes.MsgCreateClaim
		expectedErr error
	}{
		{
			desc: "onchain session ID must match claim msg session ID",
			claimMsgFn: func(t *testing.T) *prooftypes.MsgCreateClaim {
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
				prooftypes.ErrProofInvalidSessionId.Wrapf(
					"session ID does not match onchain session ID; expected %q, got %q",
					sessionRes.GetSession().GetSessionId(),
					"invalid_session_id",
				).Error(),
			),
		},
		{
			desc: "claim msg supplier operator address must be in the session",
			claimMsgFn: func(t *testing.T) *prooftypes.MsgCreateClaim {
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
				prooftypes.ErrProofNotFound.Wrapf(
					"supplier operator address %q not found in session ID %q",
					wrongSupplierOperatorAddr,
					sessionRes.GetSession().GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "claim msg supplier operator address must exist onchain",
			claimMsgFn: func(t *testing.T) *prooftypes.MsgCreateClaim {
				return newTestClaimMsg(t,
					sessionStartHeight,
					sessionRes.GetSession().GetSessionId(),
					// Use a supplier operat address that's nonexistent onchain.
					randSupplierOperatorAddr,
					appAddr,
					service,
					defaultMerkleRoot,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				prooftypes.ErrProofNotFound.Wrapf(
					"supplier operator address %q not found in session ID %q",
					randSupplierOperatorAddr,
					sessionRes.GetSession().GetSessionId(),
				).Error(),
			),
		},
		{
			desc: "claim msg application address must be in the session",
			claimMsgFn: func(t *testing.T) *prooftypes.MsgCreateClaim {
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
			desc: "claim msg application address must exist onchain",
			claimMsgFn: func(t *testing.T) *prooftypes.MsgCreateClaim {
				return newTestClaimMsg(t,
					sessionStartHeight,
					sessionRes.GetSession().GetSessionId(),
					supplierOperatorAddr,
					// Use an application address that's nonexistent onchain.
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
		{
			desc: "claim msg merkle root must have non-zero relays",
			claimMsgFn: func(t *testing.T) *prooftypes.MsgCreateClaim {
				return newTestClaimMsg(t,
					sessionStartHeight,
					sessionRes.GetSession().GetSessionId(),
					supplierOperatorAddr,
					appAddr,
					service,
					invalidRootHashWithZeroCount,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				prooftypes.ErrProofInvalidClaimRootHash.Wrapf(
					"has zero count in Merkle root (hex) %x",
					invalidRootHashWithZeroCount,
				).Error(),
			),
		},
		{
			desc: "claim msg merkle root must have non-zero compute units",
			claimMsgFn: func(t *testing.T) *prooftypes.MsgCreateClaim {
				return newTestClaimMsg(t,
					sessionStartHeight,
					sessionRes.GetSession().GetSessionId(),
					supplierOperatorAddr,
					appAddr,
					service,
					invalidRootHashWithZeroSum,
				)
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				prooftypes.ErrProofInvalidClaimRootHash.Wrapf(
					"has zero sum in Merkle root (hex) %x",
					invalidRootHashWithZeroSum,
				).Error(),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			_, err := srv.CreateClaim(ctx, test.claimMsgFn(t))
			require.ErrorContains(t, err, test.expectedErr.Error())

			// Assert that no events were emitted.
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			events := sdkCtx.EventManager().Events()
			claimCreatedEvents := testutilevents.FilterEvents[*prooftypes.EventClaimCreated](t, events)
			require.Len(t, claimCreatedEvents, 0)
		})
	}
}

func TestMsgServer_CreateClaim_Error_ComputeUnitsMismatch(t *testing.T) {
	// Set block height to 1 so there is a valid session onchain.
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
		OwnerAddress:         sample.AccAddressBech32(),
	}
	// Add the service that is expected to be onchain.
	keepers.SetService(ctx, *service)

	// Add a supplier that is expected to be in the session.
	// supplierAddr is staked for "svc1" such that it is expected to be in the session.
	supplierKeeper := keepers.SupplierKeeper
	supplierAddr := sample.AccAddressBech32()
	supplierServices := []*sharedtypes.SupplierServiceConfig{
		{ServiceId: service.Id},
	}
	serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(supplierAddr, supplierServices, 1, 0)
	supplierKeeper.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{
		OperatorAddress:      supplierAddr,
		Services:             supplierServices,
		ServiceConfigHistory: serviceConfigHistory,
	})

	// Add an application that is expected to be in the session.
	// appAddr is staked for "svc1" such that it is expected to be in the session.
	appKeeper := keepers.ApplicationKeeper
	appAddr := sample.AccAddressBech32()
	appKeeper.SetApplication(ctx, apptypes.Application{
		Address: appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: service.Id},
		},
	})

	// Query for the session which contains the expected app and supplier pair.
	sessionRes, err := keepers.GetSession(
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
	testClaim := prooftypes.Claim{RootHash: testClaimMsg.GetRootHash()}
	testClaimNumComputeUnits, err := testClaim.GetNumClaimedComputeUnits()
	require.NoError(t, err)
	testClaimNumRelays, err := testClaim.GetNumRelays()
	require.NoError(t, err)

	// Ensure that submitting the claim fails because the number of compute units
	// claimed does not match the expected amount as a function of (relay, service_CUPR)
	_, err = srv.CreateClaim(ctx, testClaimMsg)
	require.ErrorContains(t,
		err,
		status.Error(
			codes.InvalidArgument,
			prooftypes.ErrProofComputeUnitsMismatch.Wrapf(
				"claim compute units: %d is not equal to number of relays %d * compute units per relay %d for service %s",
				testClaimNumComputeUnits,
				testClaimNumRelays,
				nonDefaultComputeUnitsPerRelay,
				sessionHeader.ServiceId,
			).Error(),
		).Error(),
	)

	// Assert that no events were emitted.
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx)
	events := sdkCtx.EventManager().Events()
	claimCreatedEvents := testutilevents.FilterEvents[*prooftypes.EventClaimCreated](t, events)
	require.Len(t, claimCreatedEvents, 0)
}

func TestMsgServer_CreateClaim_SessionHeaderPreservation(t *testing.T) {
	// This test verifies that claims preserve the session header from the message,
	// not from the queried session. This is critical for historical claims where
	// parameters may have changed since the session occurred.
	blockHeight := int64(1)
	keepers, ctx := keepertest.NewProofModuleKeepers(t, keepertest.WithBlockHeight(blockHeight))
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)

	supplierOperatorAddr := sample.AccAddressBech32()
	appAddr := sample.AccAddressBech32()
	service := &sharedtypes.Service{
		Id:                   testServiceId,
		ComputeUnitsPerRelay: computeUnitsPerRelay,
		OwnerAddress:         sample.AccAddressBech32(),
	}

	supplierServices := []*sharedtypes.SupplierServiceConfig{{ServiceId: service.Id}}
	serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(supplierOperatorAddr, supplierServices, 1, 0)
	keepers.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{
		OperatorAddress:      supplierOperatorAddr,
		Services:             supplierServices,
		ServiceConfigHistory: serviceConfigHistory,
	})

	keepers.SetApplication(ctx, apptypes.Application{
		Address:        appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	})

	keepers.SetService(ctx, *service)

	sessionRes, err := keepers.GetSession(ctx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddr,
		ServiceId:          service.Id,
		BlockHeight:        blockHeight,
	})
	require.NoError(t, err)

	sessionHeader := sessionRes.GetSession().GetHeader()
	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	claimHeight := sharedtypes.GetClaimWindowCloseHeight(&sharedParams, sessionHeader.GetSessionEndBlockHeight())

	sdkCtx = sdkCtx.WithBlockHeight(claimHeight)
	ctx = sdkCtx

	// Create a custom session header with specific values that we want to preserve
	customSessionHeader := &sessiontypes.SessionHeader{
		ApplicationAddress:      sessionHeader.ApplicationAddress,
		ServiceId:               sessionHeader.ServiceId,
		SessionId:               sessionHeader.SessionId,
		SessionStartBlockHeight: sessionHeader.SessionStartBlockHeight,
		SessionEndBlockHeight:   sessionHeader.SessionEndBlockHeight,
	}

	claimMsg := prooftypes.NewMsgCreateClaim(
		supplierOperatorAddr,
		customSessionHeader,
		defaultMerkleRoot,
	)

	_, err = srv.CreateClaim(ctx, claimMsg)
	require.NoError(t, err)

	// Verify the claim was stored with the exact session header from the message
	claimRes, err := keepers.AllClaims(ctx, &prooftypes.QueryAllClaimsRequest{})
	require.NoError(t, err)
	require.Len(t, claimRes.GetClaims(), 1)

	storedClaim := claimRes.GetClaims()[0]
	storedHeader := storedClaim.GetSessionHeader()

	// CRITICAL: The stored claim must have the session header from the message,
	// not from the queried session. This ensures historical accuracy.
	require.Equal(t, customSessionHeader.SessionId, storedHeader.SessionId,
		"claim should preserve session ID from message")
	require.Equal(t, customSessionHeader.ApplicationAddress, storedHeader.ApplicationAddress,
		"claim should preserve application address from message")
	require.Equal(t, customSessionHeader.ServiceId, storedHeader.ServiceId,
		"claim should preserve service ID from message")
	require.Equal(t, customSessionHeader.SessionStartBlockHeight, storedHeader.SessionStartBlockHeight,
		"claim should preserve session start height from message")
	require.Equal(t, customSessionHeader.SessionEndBlockHeight, storedHeader.SessionEndBlockHeight,
		"claim should preserve session end height from message")
}

func TestMsgServer_CreateClaim_SessionAlignmentValidation(t *testing.T) {
	// This test verifies that claims with mismatched session start/end block heights
	// are rejected to ensure session metadata correctness.
	blockHeight := int64(1)
	keepers, ctx := keepertest.NewProofModuleKeepers(t, keepertest.WithBlockHeight(blockHeight))
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)

	supplierOperatorAddr := sample.AccAddressBech32()
	appAddr := sample.AccAddressBech32()
	service := &sharedtypes.Service{
		Id:                   testServiceId,
		ComputeUnitsPerRelay: computeUnitsPerRelay,
		OwnerAddress:         sample.AccAddressBech32(),
	}

	supplierServices := []*sharedtypes.SupplierServiceConfig{{ServiceId: service.Id}}
	serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(supplierOperatorAddr, supplierServices, 1, 0)
	keepers.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{
		OperatorAddress:      supplierOperatorAddr,
		Services:             supplierServices,
		ServiceConfigHistory: serviceConfigHistory,
	})

	keepers.SetApplication(ctx, apptypes.Application{
		Address:        appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
	})

	keepers.SetService(ctx, *service)

	sessionRes, err := keepers.GetSession(ctx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddr,
		ServiceId:          service.Id,
		BlockHeight:        blockHeight,
	})
	require.NoError(t, err)

	sessionHeader := sessionRes.GetSession().GetHeader()
	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	claimHeight := sharedtypes.GetClaimWindowCloseHeight(&sharedParams, sessionHeader.GetSessionEndBlockHeight())

	sdkCtx = sdkCtx.WithBlockHeight(claimHeight)
	ctx = sdkCtx

	tests := []struct {
		desc                    string
		sessionStartBlockHeight int64
		sessionEndBlockHeight   int64
		expectedErrContains     string
	}{
		{
			desc:                    "mismatched session start block height (incremented)",
			sessionStartBlockHeight: sessionHeader.SessionStartBlockHeight + 1,
			sessionEndBlockHeight:   sessionHeader.SessionEndBlockHeight + 1,
			expectedErrContains:     "session start block height does not match",
		},
		{
			desc:                    "mismatched session end block height",
			sessionStartBlockHeight: sessionHeader.SessionStartBlockHeight,
			sessionEndBlockHeight:   sessionHeader.SessionEndBlockHeight + 1,
			expectedErrContains:     "session end block height does not match",
		},
		{
			desc:                    "both start and end heights mismatched by different offsets",
			sessionStartBlockHeight: sessionHeader.SessionStartBlockHeight + 2,
			sessionEndBlockHeight:   sessionHeader.SessionEndBlockHeight + 3,
			expectedErrContains:     "session start block height does not match",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			invalidHeader := &sessiontypes.SessionHeader{
				ApplicationAddress:      sessionHeader.ApplicationAddress,
				ServiceId:               sessionHeader.ServiceId,
				SessionId:               sessionHeader.SessionId,
				SessionStartBlockHeight: test.sessionStartBlockHeight,
				SessionEndBlockHeight:   test.sessionEndBlockHeight,
			}

			claimMsg := prooftypes.NewMsgCreateClaim(
				supplierOperatorAddr,
				invalidHeader,
				defaultMerkleRoot,
			)

			_, err := srv.CreateClaim(ctx, claimMsg)
			require.Error(t, err)
			require.ErrorContains(t, err, test.expectedErrContains,
				"should reject claim with mismatched session heights")
		})
	}
}

func newTestClaimMsg(
	t *testing.T,
	sessionStartHeight int64,
	sessionId string,
	supplierOperatorAddr string,
	appAddr string,
	service *sharedtypes.Service,
	merkleRoot smt.MerkleSumRoot,
) *prooftypes.MsgCreateClaim {
	t.Helper()

	return prooftypes.NewMsgCreateClaim(
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

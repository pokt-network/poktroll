package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestMsgServer_CreateClaim_CuprPinnedToSessionStart is a regression test for the
// mid-session compute_units_per_relay (cupr) forfeit bug: a service that changed its
// cupr while sessions were open rejected every in-flight claim with
// ErrProofComputeUnitsMismatch, because claim validation read the LIVE cupr while the
// RelayMiner had baked the mine-time (session-start) cupr into the append-only SMST.
//
// Claim validation now pins cupr to the session-start height, so:
//   - a claim built at the session-start cupr is accepted even though the live cupr
//     changed mid-session (the bug scenario — this previously 1126'd), and
//   - a claim built at the NEW cupr for a session that started BEFORE the change is
//     rejected (proving validation uses session-start, not live).
func TestMsgServer_CreateClaim_CuprPinnedToSessionStart(t *testing.T) {
	const (
		oldCupr = uint64(2)
		newCupr = uint64(5)
	)

	rootOldCupr := testproof.SmstRootWithSumAndCount(expectedNumRelays*oldCupr, expectedNumRelays)
	rootNewCupr := testproof.SmstRootWithSumAndCount(expectedNumRelays*newCupr, expectedNumRelays)

	type cuprTestEnv struct {
		keepers              *keepertest.ProofModuleKeepers
		claimCtx             cosmostypes.Context
		supplierOperatorAddr string
		appAddr              string
		service              *sharedtypes.Service
		sessionHeader        *sessiontypes.SessionHeader
	}

	// setupMidSessionCuprChange opens a session at height 1 with cupr=oldCupr, simulates
	// a mid-session change to newCupr (live cupr + recorded history), and returns an
	// environment positioned at a valid claim height (claim window close).
	setupMidSessionCuprChange := func(t *testing.T) cuprTestEnv {
		t.Helper()

		supplierOperatorAddr := sample.AccAddressBech32()
		blockHeight := int64(1)
		keepers, ctx := keepertest.NewProofModuleKeepers(t, keepertest.WithBlockHeight(blockHeight))
		sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

		service := &sharedtypes.Service{
			Id:                   testServiceId,
			ComputeUnitsPerRelay: oldCupr,
			OwnerAddress:         sample.AccAddressBech32(),
		}
		appAddr := sample.AccAddressBech32()

		supplierServices := []*sharedtypes.SupplierServiceConfig{{ServiceId: service.Id}}
		serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
			supplierOperatorAddr, supplierServices, 1, 0,
		)
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

		sessionRes, err := keepers.GetSession(ctx, &sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			ServiceId:          service.Id,
			BlockHeight:        blockHeight,
		})
		require.NoError(t, err)
		sessionHeader := sessionRes.GetSession().GetHeader()

		// Simulate a mid-session cupr change: the live cupr becomes newCupr and the
		// change is recorded in history. The in-flight session (start height 1) must
		// still resolve to oldCupr.
		svcKeeper := keepers.ServiceKeeper.(*servicekeeper.Keeper)
		midSessionCtx := sdkCtx.WithBlockHeight(blockHeight + 1)
		require.NoError(t, svcKeeper.SnapshotServiceComputeUnitsPerRelayChange(
			midSessionCtx, service.Id, oldCupr, newCupr,
		))
		updatedService := *service
		updatedService.ComputeUnitsPerRelay = newCupr
		keepers.SetService(midSessionCtx, updatedService)

		// Sanity: session-start cupr is still the OLD value even though live is new.
		cuprAtStart, found := svcKeeper.GetServiceComputeUnitsPerRelayAtHeight(
			midSessionCtx, service.Id, sessionHeader.GetSessionStartBlockHeight(),
		)
		require.True(t, found)
		require.Equal(t, oldCupr, cuprAtStart)

		// Advance to a valid claim height (claim window close).
		sharedParams := keepers.SharedKeeper.GetParams(ctx)
		claimHeight := sharedtypes.GetClaimWindowCloseHeight(&sharedParams, sessionHeader.GetSessionEndBlockHeight())
		claimCtx := sdkCtx.WithBlockHeight(claimHeight)

		return cuprTestEnv{
			keepers:              keepers,
			claimCtx:             claimCtx,
			supplierOperatorAddr: supplierOperatorAddr,
			appAddr:              appAddr,
			service:              service,
			sessionHeader:        sessionHeader,
		}
	}

	t.Run("claim built at session-start cupr is accepted despite live cupr change", func(t *testing.T) {
		env := setupMidSessionCuprChange(t)
		srv := keeper.NewMsgServerImpl(*env.keepers.Keeper)

		claimMsg := newTestClaimMsg(t,
			env.sessionHeader.GetSessionStartBlockHeight(),
			env.sessionHeader.GetSessionId(),
			env.supplierOperatorAddr,
			env.appAddr,
			env.service,
			rootOldCupr,
		)

		_, err := srv.CreateClaim(env.claimCtx, claimMsg)
		require.NoError(t, err, "claim built at the session-start cupr must be accepted after a mid-session cupr change")
	})

	t.Run("claim built at the new (live) cupr is rejected for a pre-change session", func(t *testing.T) {
		env := setupMidSessionCuprChange(t)
		srv := keeper.NewMsgServerImpl(*env.keepers.Keeper)

		claimMsg := newTestClaimMsg(t,
			env.sessionHeader.GetSessionStartBlockHeight(),
			env.sessionHeader.GetSessionId(),
			env.supplierOperatorAddr,
			env.appAddr,
			env.service,
			rootNewCupr,
		)

		_, err := srv.CreateClaim(env.claimCtx, claimMsg)
		require.Error(t, err, "claim built at the new cupr must be rejected because validation pins to the session-start cupr")
		require.ErrorContains(t, err, "compute units")
	})
}

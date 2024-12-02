package application

import (
	"context"
	"testing"

	cosmoslog "cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	testevents "github.com/pokt-network/poktroll/testutil/events"
	"github.com/pokt-network/poktroll/testutil/keeper"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
)

type applicationMinStakeTestSuite struct {
	suite.Suite

	ctx     context.Context
	keepers keeper.TokenomicsModuleKeepers

	serviceId,
	appBech32,
	supplierBech32 string

	appStake          *cosmostypes.Coin
	appServiceConfigs []*sharedtypes.ApplicationServiceConfig

	numRelays,
	numComputeUnitsPerRelay uint64
}

func TestApplicationMinStakeTestSuite(t *testing.T) {
	cmd.InitSDKConfig()

	suite.Run(t, new(applicationMinStakeTestSuite))
}

func (s *applicationMinStakeTestSuite) SetupTest() {
	s.keepers, s.ctx = keeper.NewTokenomicsModuleKeepers(s.T(),
		cosmoslog.NewNopLogger(),
		keeper.WithProofRequirement(false),
		keeper.WithDefaultModuleBalances(),
	)

	proofParams := prooftypes.DefaultParams()
	proofParams.ProofRequestProbability = 0
	err := s.keepers.ProofKeeper.SetParams(s.ctx, proofParams)
	require.NoError(s.T(), err)

	s.serviceId = "svc1"
	s.appBech32 = sample.AccAddress()
	s.supplierBech32 = sample.AccAddress()
	s.numRelays = 10
	s.numComputeUnitsPerRelay = 1

	s.appStake = &apptypes.DefaultMinStake
	s.appServiceConfigs = []*sharedtypes.ApplicationServiceConfig{{ServiceId: s.serviceId}}

	// Set block height to 1.
	s.ctx = cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(1)
}

func (s *applicationMinStakeTestSuite) TestAppCannotStakeLessThanMinStake() {
	s.T().Skip("this case is well covered in x/application/keeper/msg_server_stake_application_test.go")
}

func (s *applicationMinStakeTestSuite) TestAppIsUnbondedIfBelowMinStakeWhenSettling() {
	// Assert that the application's initial bank balance is 0.
	appBalance := s.getAppBalance()
	require.Equal(s.T(), int64(0), appBalance.Amount.Int64())

	// Add service 1
	s.addService()

	// Stake an application for service 1 with min stake.
	s.stakeApp()

	// Stake a supplier for service 1.
	s.stakeSupplier()

	// Get the session header.
	sessionHeader := s.getSessionHeader()

	// Create a claim whose settlement amount drops the application below min stake
	claim := s.getClaim(sessionHeader)
	s.keepers.ProofKeeper.UpsertClaim(s.ctx, *claim)

	// Set the current height to the claim settlement session end height.
	sharedParams := s.keepers.SharedKeeper.GetParams(s.ctx)
	settlementSessionEndHeight := sharedtypes.GetSettlementSessionEndHeight(&sharedParams, s.getCurrentHeight())
	s.setBlockHeight(settlementSessionEndHeight)

	// Settle pending claims; this should cause the application to be unbonded.
	_, _, err := s.keepers.Keeper.SettlePendingClaims(cosmostypes.UnwrapSDKContext(s.ctx))
	require.NoError(s.T(), err)

	expectedApp := s.getExpectedApp(claim)

	// Assert that the EventApplicationUnbondingBegin event is emitted.
	s.assertUnbondingBeginEventObserved(expectedApp)

	// Reset the events, as if a new block were created.
	s.ctx, _ = testevents.ResetEventManager(s.ctx)

	// Set the current height to the unbonding session end height.
	unbondingSessionEndHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, expectedApp)
	s.setBlockHeight(unbondingSessionEndHeight)

	// Run app module end blockers to complete unbonding.
	err = s.keepers.ApplicationKeeper.EndBlockerUnbondApplications(s.ctx)
	require.NoError(s.T(), err)

	// Assert that the EventApplicationUnbondingEnd event is emitted.
	s.assertUnbondingEndEventObserved(expectedApp)

	// Assert that the application was unbonded.
	_, isAppFound := s.keepers.ApplicationKeeper.GetApplication(s.ctx, s.appBech32)
	require.False(s.T(), isAppFound)

	// Assert that the application's stake was returned to its bank balance.
	s.assertAppStakeIsReturnedToBalance()
}

// addService adds the test service to the service module state.
func (s *applicationMinStakeTestSuite) addService() {
	s.keepers.ServiceKeeper.SetService(s.ctx, sharedtypes.Service{
		Id:                   s.serviceId,
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddress(), // random address.
	})
}

// stakeApp stakes an application for service 1 with min stake.
func (s *applicationMinStakeTestSuite) stakeApp() {
	s.keepers.ApplicationKeeper.SetApplication(s.ctx, apptypes.Application{
		Address:        s.appBech32,
		Stake:          s.appStake,
		ServiceConfigs: s.appServiceConfigs,
	})
}

// stakeSupplier stakes a supplier for service 1.
func (s *applicationMinStakeTestSuite) stakeSupplier() {
	s.keepers.SupplierKeeper.SetSupplier(s.ctx, sharedtypes.Supplier{
		OwnerAddress:    s.supplierBech32,
		OperatorAddress: s.supplierBech32,
		Stake:           &suppliertypes.DefaultMinStake,
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: s.serviceId,
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{
						Address:            s.supplierBech32,
						RevSharePercentage: 100,
					},
				},
			},
		},
	})
}

// getSessionHeader gets the session header for the test session.
func (s *applicationMinStakeTestSuite) getSessionHeader() *sessiontypes.SessionHeader {
	s.T().Helper()

	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx)
	currentHeight := sdkCtx.BlockHeight()
	sessionRes, err := s.keepers.SessionKeeper.GetSession(s.ctx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: s.appBech32,
		ServiceId:          s.serviceId,
		BlockHeight:        currentHeight,
	})
	require.NoError(s.T(), err)

	return sessionRes.GetSession().GetHeader()
}

// getClaim creates a claim whose settlement amount drops the application below min stake.
func (s *applicationMinStakeTestSuite) getClaim(
	sessionHeader *sessiontypes.SessionHeader,
) *prooftypes.Claim {
	claimRoot := testproof.SmstRootWithSumAndCount(s.numRelays*s.numComputeUnitsPerRelay, s.numRelays)

	return &prooftypes.Claim{
		SupplierOperatorAddress: s.supplierBech32,
		SessionHeader:           sessionHeader,
		RootHash:                claimRoot,
	}
}

// getAppBalance returns the bank module balance for the application.
func (s *applicationMinStakeTestSuite) getAppBalance() *cosmostypes.Coin {
	s.T().Helper()

	appBalRes, err := s.keepers.BankKeeper.Balance(s.ctx, &banktypes.QueryBalanceRequest{
		Address: s.appBech32, Denom: volatile.DenomuPOKT,
	})
	require.NoError(s.T(), err)

	return appBalRes.GetBalance()
}

// getCurrentHeight gets the current height from the context.
func (s *applicationMinStakeTestSuite) getCurrentHeight() int64 {
	return cosmostypes.UnwrapSDKContext(s.ctx).BlockHeight()
}

// setBlockHeight sets the current block height in the context to targetHeight.
func (s *applicationMinStakeTestSuite) setBlockHeight(targetHeight int64) cosmostypes.Context {
	sdkCtx := cosmostypes.
		UnwrapSDKContext(s.ctx).
		WithBlockHeight(targetHeight)
	s.ctx = sdkCtx
	return sdkCtx
}

// getExpectedApp returns the expected application for the given claim.
func (s *applicationMinStakeTestSuite) getExpectedApp(claim *prooftypes.Claim) *apptypes.Application {
	s.T().Helper()

	sharedParams := s.keepers.SharedKeeper.GetParams(s.ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, s.getCurrentHeight())
	relayMiningDifficulty := s.newRelayminingDifficulty()
	expectedBurnCoin, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(s.T(), err)

	globalInflationPerClaim := s.keepers.Keeper.GetParams(s.ctx).GlobalInflationPerClaim
	globalInflationAmt, _ := tlm.CalculateGlobalPerClaimMintInflationFromSettlementAmount(expectedBurnCoin, globalInflationPerClaim)
	expectedEndStake := s.appStake.Sub(expectedBurnCoin).Sub(globalInflationAmt)
	return &apptypes.Application{
		Address:                   s.appBech32,
		Stake:                     &expectedEndStake,
		ServiceConfigs:            s.appServiceConfigs,
		DelegateeGatewayAddresses: make([]string, 0),
		PendingUndelegations:      make(map[uint64]apptypes.UndelegatingGatewayList),
		UnstakeSessionEndHeight:   uint64(sessionEndHeight),
	}
}

// newRelayminingDifficulty creates a new RelayMiningDifficulty for use in calculating application burn.
func (s *applicationMinStakeTestSuite) newRelayminingDifficulty() servicetypes.RelayMiningDifficulty {
	s.T().Helper()

	targetNumRelays := s.keepers.ServiceKeeper.GetParams(s.ctx).TargetNumRelays

	return servicekeeper.NewDefaultRelayMiningDifficulty(
		s.ctx,
		cosmoslog.NewNopLogger(),
		s.serviceId,
		targetNumRelays,
		targetNumRelays,
	)
}

// assertUnbondingBeginEventObserved asserts that the EventApplicationUnbondingBegin
// event is emitted and matches one derived from the given expected application.
func (s *applicationMinStakeTestSuite) assertUnbondingBeginEventObserved(expectedApp *apptypes.Application) {
	s.T().Helper()

	sharedParams := s.keepers.SharedKeeper.GetParams(s.ctx)
	unbondingEndHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, expectedApp)
	sessionEndHeight := s.keepers.SharedKeeper.GetSessionEndHeight(s.ctx, s.getCurrentHeight())
	expectedAppUnbondingBeginEvent := &apptypes.EventApplicationUnbondingBegin{
		Application:        expectedApp,
		Reason:             apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_BELOW_MIN_STAKE,
		SessionEndHeight:   sessionEndHeight,
		UnbondingEndHeight: unbondingEndHeight,
	}

	events := cosmostypes.UnwrapSDKContext(s.ctx).EventManager().Events()
	appUnbondingBeginEvents := testevents.FilterEvents[*apptypes.EventApplicationUnbondingBegin](s.T(), events)
	require.Equal(s.T(), 1, len(appUnbondingBeginEvents), "expected exactly 1 event")
	require.EqualValues(s.T(), expectedAppUnbondingBeginEvent, appUnbondingBeginEvents[0])
}

// assertUnbondingEndEventObserved asserts that the EventApplicationUnbondingEnd is
// emitted and matches one derived from the given expected application.
func (s *applicationMinStakeTestSuite) assertUnbondingEndEventObserved(expectedApp *apptypes.Application) {
	s.T().Helper()

	sharedParams := s.keepers.SharedKeeper.GetParams(s.ctx)
	unbondingEndHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, expectedApp)
	unbondingSessionEndHeight := apptypes.GetApplicationUnbondingHeight(&sharedParams, expectedApp)
	expectedAppUnbondingEndEvent := &apptypes.EventApplicationUnbondingEnd{
		Application:        expectedApp,
		Reason:             apptypes.ApplicationUnbondingReason_APPLICATION_UNBONDING_REASON_BELOW_MIN_STAKE,
		SessionEndHeight:   unbondingSessionEndHeight,
		UnbondingEndHeight: unbondingEndHeight,
	}

	events := cosmostypes.UnwrapSDKContext(s.ctx).EventManager().Events()
	appUnbondingEndEvents := testevents.FilterEvents[*apptypes.EventApplicationUnbondingEnd](s.T(), events)
	require.Equal(s.T(), 1, len(appUnbondingEndEvents), "expected exactly 1 event")
	require.EqualValues(s.T(), expectedAppUnbondingEndEvent, appUnbondingEndEvents[0])
}

// assertAppStakeIsReturnedToBalance asserts that the application's stake is returned to its bank balance.
func (s *applicationMinStakeTestSuite) assertAppStakeIsReturnedToBalance() {
	s.T().Helper()

	expectedAppBurn := int64(s.numRelays * s.numComputeUnitsPerRelay * sharedtypes.DefaultComputeUnitsToTokensMultiplier)
	expectedAppBurnCoin := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, expectedAppBurn)
	globalInflationPerClaim := s.keepers.Keeper.GetParams(s.ctx).GlobalInflationPerClaim
	globalInflationCoin, _ := tlm.CalculateGlobalPerClaimMintInflationFromSettlementAmount(expectedAppBurnCoin, globalInflationPerClaim)
	expectedAppBalance := s.appStake.Sub(expectedAppBurnCoin).Sub(globalInflationCoin)

	appBalance := s.getAppBalance()
	require.Equal(s.T(), expectedAppBalance.Amount.Int64(), appBalance.Amount.Int64())
}

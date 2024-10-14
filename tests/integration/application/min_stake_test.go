package application

import (
	"context"
	"math"
	"testing"

	cosmoslog "cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/keeper"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
)

type applicationMinStakeTestSuite struct {
	suite.Suite

	ctx     context.Context
	keepers keeper.TokenomicsModuleKeepers

	serviceId,
	appBech32,
	supplierBech32 string

	appStake *cosmostypes.Coin

	numRelays,
	numComputeUnitsPerRelay uint64
}

func TestApplicationMinStakeTestSuite(t *testing.T) {
	cmd.InitSDKConfig()

	suite.Run(t, new(applicationMinStakeTestSuite))
}

func (s *applicationMinStakeTestSuite) SetupTest() {
	s.keepers, s.ctx = keeper.NewTokenomicsModuleKeepers(s.T(), cosmoslog.NewNopLogger())

	proofParams := prooftypes.DefaultParams()
	proofParams.ProofRequestProbability = 0
	err := s.keepers.ProofKeeper.SetParams(s.ctx, proofParams)
	require.NoError(s.T(), err)

	s.serviceId = "svc1"
	s.appBech32 = sample.AccAddress()
	s.supplierBech32 = sample.AccAddress()
	s.appStake = &apptypes.DefaultMinStake
	s.numRelays = 10
	s.numComputeUnitsPerRelay = 1

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

	proofParams := s.keepers.ProofKeeper.GetParams(s.ctx)
	proofParams.ProofRequestProbability = 0
	proofRequirementThreshold := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, math.MaxInt64)
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	s.keepers.ProofKeeper.SetParams(s.ctx, proofParams)

	// Get the session header.
	sessionHeader := s.getSessionHeader()

	// Create a claim whose settlement amount drops the application below min stake
	claim := s.getClaim(sessionHeader)
	s.keepers.ProofKeeper.UpsertClaim(s.ctx, *claim)

	// Set the current height to the claim settlement height.
	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx)
	currentHeight := sdkCtx.BlockHeight()
	sharedParams := s.keepers.SharedKeeper.GetParams(s.ctx)
	currentSessionEndHeight := shared.GetSessionEndHeight(&sharedParams, currentHeight)
	claimSettlementHeight := currentSessionEndHeight + int64(sharedtypes.GetSessionEndToProofWindowCloseBlocks(&sharedParams)) + 1
	sdkCtx = sdkCtx.WithBlockHeight(claimSettlementHeight)
	s.ctx = sdkCtx

	// Settle pending claims; this should cause the application to be unbonded.
	_, _, err := s.keepers.Keeper.SettlePendingClaims(sdkCtx)
	require.NoError(s.T(), err)

	// Assert that the application was unbonded.
	_, isAppFound := s.keepers.ApplicationKeeper.GetApplication(s.ctx, s.appBech32)
	require.False(s.T(), isAppFound)

	// Assert that the remaining application's stake was returned to its bank balance.
	expectedAppBurn := sdkmath.NewInt(int64(s.numRelays * s.numComputeUnitsPerRelay * sharedtypes.DefaultComputeUnitsToTokensMultiplier))
	globalInflationAmount := float64(expectedAppBurn.Uint64()) * tokenomicskeeper.MintPerClaimedTokenGlobalInflation
	globalInflationAmountInt := sdkmath.NewInt(int64(globalInflationAmount))
	expectedAppBalance := s.appStake.SubAmount(expectedAppBurn).SubAmount(globalInflationAmountInt)
	appBalance = s.getAppBalance()
	require.Equal(s.T(), expectedAppBalance.Amount.Int64(), appBalance.Amount.Int64())

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
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: s.serviceId}},
	})
}

// stakeSupplier stakes a supplier for service 1.
func (s *applicationMinStakeTestSuite) stakeSupplier() {
	// TODO_UPNEXT(@bryanchriswhite, #612): Replace supplierStake with suppleirtypes.DefaultMinStake.
	supplierStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000000) // 1 POKT.
	s.keepers.SupplierKeeper.SetSupplier(s.ctx, sharedtypes.Supplier{
		OwnerAddress:    s.supplierBech32,
		OperatorAddress: s.supplierBech32,
		Stake:           &supplierStake,
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
	appBalRes, err := s.keepers.BankKeeper.Balance(s.ctx, &banktypes.QueryBalanceRequest{
		Address: s.appBech32, Denom: volatile.DenomuPOKT,
	})
	require.NoError(s.T(), err)

	return appBalRes.GetBalance()
}

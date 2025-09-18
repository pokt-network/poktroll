package token_logic_modules

import (
	"context"
	"math"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

type tokenLogicModuleTestSuite struct {
	suite.Suite

	ctx     context.Context
	keepers testkeeper.TokenomicsModuleKeepers

	service  *sharedtypes.Service
	app      *apptypes.Application
	supplier *sharedtypes.Supplier

	proposerConsAddr        string
	proposerValOperatorAddr string
	sourceOwnerAddr         string
	daoRewardAddr           string

	expectedSettledResults,
	expectedExpiredResults tlm.ClaimSettlementResults
	expectedSettlementState *settlementState
}

// settlementState holds the expected post-settlement app stake and rewardee balances.
type settlementState struct {
	appModuleBalance        *cosmostypes.Coin
	supplierModuleBalance   *cosmostypes.Coin
	tokenomicsModuleBalance *cosmostypes.Coin

	appStake             *cosmostypes.Coin
	supplierOwnerBalance *cosmostypes.Coin
	proposerBalance      *cosmostypes.Coin
	daoBalance           *cosmostypes.Coin
	sourceOwnerBalance   *cosmostypes.Coin
}

func init() {
	cmd.InitSDKConfig()
}

func TestTLMProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(tokenLogicModuleTestSuite))
}

// SetupTest generates and sets all rewardee addresses on the suite, and
// set a service, application, and supplier on the suite.
func (s *tokenLogicModuleTestSuite) SetupTest() {
	s.daoRewardAddr = sample.AccAddressBech32()
	s.sourceOwnerAddr = sample.AccAddressBech32()
	s.proposerConsAddr = sample.ConsAddressBech32()
	s.proposerValOperatorAddr = sample.ValOperatorAddressBech32()

	s.service = &sharedtypes.Service{
		Id:                   "svc1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         s.sourceOwnerAddr,
	}

	appStake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, math.MaxInt64)
	s.app = &apptypes.Application{
		Address: sample.AccAddressBech32(),
		Stake:   &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: s.service.GetId()},
		},
	}

	supplierBech32 := sample.AccAddressBech32()
	services := []*sharedtypes.SupplierServiceConfig{
		{
			ServiceId: s.service.GetId(),
			RevShare: []*sharedtypes.ServiceRevenueShare{
				{
					Address:            supplierBech32,
					RevSharePercentage: 100,
				},
			},
		},
	}
	serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(supplierBech32, services, 1, 0)
	s.supplier = &sharedtypes.Supplier{
		OwnerAddress:         supplierBech32,
		OperatorAddress:      supplierBech32,
		Stake:                &suppliertypes.DefaultMinStake,
		Services:             services,
		ServiceConfigHistory: serviceConfigHistory,
	}
}

// getProofParams returns the default proof params with a high proof requirement threshold
// and no proof request probability such that no claims require a proof.
func (s *tokenLogicModuleTestSuite) getProofParams() *prooftypes.Params {
	proofParams := prooftypes.DefaultParams()
	highProofRequirementThreshold := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, math.MaxInt64)
	proofParams.ProofRequirementThreshold = &highProofRequirementThreshold
	proofParams.ProofRequestProbability = 0
	return &proofParams
}

// getSharedParams returns the default shared params with the CUTTM set to 1.
func (s *tokenLogicModuleTestSuite) getSharedParams() *sharedtypes.Params {
	sharedParams := sharedtypes.DefaultParams()
	sharedParams.ComputeUnitsToTokensMultiplier = 1 * sharedParams.ComputeUnitCostGranularity
	return &sharedParams
}

// getTokenomicsParams returns the default tokenomics params with the dao_reward_address set to s.daoRewardAddr.
func (s *tokenLogicModuleTestSuite) getTokenomicsParams() *tokenomicstypes.Params {
	tokenomicsParams := tokenomicstypes.DefaultParams()
	tokenomicsParams.DaoRewardAddress = s.daoRewardAddr
	return &tokenomicsParams
}

// getTokenomicsParamsWithCleanValidatorMath returns tokenomics params with 10% validator allocation
// for both TLMs to ensure clean mathematical divisions in validator reward distribution tests.
func (s *tokenLogicModuleTestSuite) getTokenomicsParamsWithCleanValidatorMath() *tokenomicstypes.Params {
	tokenomicsParams := tokenomicstypes.DefaultParams()
	tokenomicsParams.DaoRewardAddress = s.daoRewardAddr

	// Set validator allocation to 10% (instead of default 5%) for clean math
	// This makes total validator rewards = 110 with our test setup, which divides
	// evenly by stake ratio sum of 11, giving clean results [50, 40, 20]
	tokenomicsParams.MintAllocationPercentages.Proposer = 0.10
	tokenomicsParams.MintEqualsBurnClaimDistribution.Proposer = 0.10

	// Adjust other percentages to maintain 100% total
	// TLMGlobalMint: DAO=0.05, Proposer=0.10, Supplier=0.70, SourceOwner=0.15, Application=0.0
	tokenomicsParams.MintAllocationPercentages.Dao = 0.05
	tokenomicsParams.MintAllocationPercentages.Supplier = 0.70
	tokenomicsParams.MintAllocationPercentages.SourceOwner = 0.15
	tokenomicsParams.MintAllocationPercentages.Application = 0.0

	// TLMRelayBurnEqualsMint: DAO=0.05, Proposer=0.10, Supplier=0.70, SourceOwner=0.15, Application=0.0
	tokenomicsParams.MintEqualsBurnClaimDistribution.Dao = 0.05
	tokenomicsParams.MintEqualsBurnClaimDistribution.Supplier = 0.70
	tokenomicsParams.MintEqualsBurnClaimDistribution.SourceOwner = 0.15
	tokenomicsParams.MintEqualsBurnClaimDistribution.Application = 0.0

	return &tokenomicsParams
}

// createClaims creates numClaims number of claims, each for a unique application address.
// This ensures that each claim represents a distinct session, avoiding UpsertClaim updating the same claim.
// DEV_NOTE: The sum/count must be large enough to avoid a proposer reward
// (or other small proportion rewards) from being truncated to zero (> 1upokt).
func (s *tokenLogicModuleTestSuite) createClaims(
	keepers *testkeeper.TokenomicsModuleKeepers,
	numClaims int,
) {
	s.T().Helper()

	// Create claims for unique applications to ensure distinct sessions
	for i := 0; i < numClaims; i++ {
		// Generate a unique application address for each claim
		uniqueAppAddr := sample.AccAddressBech32()

		// Create an application entry for this address
		uniqueApp := apptypes.Application{
			Address: uniqueAppAddr,
			Stake:   s.app.Stake, // Use same stake as default app
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
				{ServiceId: s.service.GetId()},
			},
		}
		keepers.SetApplication(s.ctx, uniqueApp)

		// Get session for this unique application
		session, err := s.keepers.GetSession(s.ctx, &sessiontypes.QueryGetSessionRequest{
			ServiceId:          s.service.GetId(),
			ApplicationAddress: uniqueAppAddr,
			BlockHeight:        1,
		})
		require.NoError(s.T(), err)

		// Create claim for this unique session
		claim := prooftypes.Claim{
			SupplierOperatorAddress: s.supplier.GetOperatorAddress(),
			SessionHeader:           session.GetSession().GetHeader(),
			RootHash:                proof.SmstRootWithSumAndCount(1000, 1000),
		}

		keepers.UpsertClaim(s.ctx, claim)
	}
}

// settleClaims sets the block height to the settlement height for the current
// session and triggers the settlement of all pending claims.
func (s *tokenLogicModuleTestSuite) settleClaims(t *testing.T) (settledResults, expiredResults tlm.ClaimSettlementResults) {
	// Increment the block height to the settlement height.
	settlementHeight := sharedtypes.GetSettlementSessionEndHeight(s.getSharedParams(), 1)
	s.setBlockHeight(settlementHeight)

	settledPendingResults, expiredPendingResults, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(cosmostypes.UnwrapSDKContext(s.ctx))
	require.NoError(t, err)

	require.NotZero(t, len(settledPendingResults))
	// TODO_IMPROVE: enhance the test scenario to include expiring claims to increase coverage.
	require.Zero(t, len(expiredPendingResults))
	require.Zero(t, numDiscardedFaultyClaims)

	return settledPendingResults, expiredPendingResults
}

// setBlockHeight sets the block height of the suite's context to height.
func (s *tokenLogicModuleTestSuite) setBlockHeight(height int64) {
	s.ctx = cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(height)
}

// assertNoPendingClaims asserts that no pending claims exist.
func (s *tokenLogicModuleTestSuite) assertNoPendingClaims(t *testing.T) {
	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx)
	logger := s.keepers.Logger().With("method", "assertNoPendingClaims")
	settlementContext := tokenomicskeeper.NewSettlementContext(sdkCtx, s.keepers.Keeper, logger)
	blockHeight := sdkCtx.BlockHeight()
	pendingClaimsIterator := s.keepers.GetExpiringClaimsIterator(sdkCtx, settlementContext, blockHeight)
	defer pendingClaimsIterator.Close()

	numExpiringClaims := 0
	for pendingClaimsIterator.Valid() {
		numExpiringClaims++
		pendingClaimsIterator.Next()
	}
	require.Zero(t, numExpiringClaims)
}

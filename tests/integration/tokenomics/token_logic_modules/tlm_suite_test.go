package token_logic_modules

import (
	"context"
	"math"
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// daoRewardBech32 is a random address intended for use in tests.
// In the commutativity test, the dao_reward_address is set to this
// address and MUST remain unchanged between permutations.
var daoRewardBech32 = sample.AccAddress()

type tokenLogicModuleTestSuite struct {
	suite.Suite

	ctx     context.Context
	keepers testkeeper.TokenomicsModuleKeepers

	service  *sharedtypes.Service
	app      *apptypes.Application
	supplier *sharedtypes.Supplier

	proposerConsAddr cosmostypes.ConsAddress
	sourceOwnerBech32,
	daoRewardBech32 string

	expectedSettledResults,
	expectedExpiredResults tlm.SettlementResults
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
	s.daoRewardBech32 = daoRewardBech32
	s.sourceOwnerBech32 = sample.AccAddress()
	s.proposerConsAddr = sample.ConsAddress()

	s.service = &sharedtypes.Service{
		Id:                   "svc1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         s.sourceOwnerBech32,
	}

	appStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, math.MaxInt64)
	s.app = &apptypes.Application{
		Address: sample.AccAddress(),
		Stake:   &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: s.service.GetId()},
		},
	}

	supplierBech32 := sample.AccAddress()
	s.supplier = &sharedtypes.Supplier{
		OwnerAddress:    supplierBech32,
		OperatorAddress: supplierBech32,
		Stake:           &suppliertypes.DefaultMinStake,
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: s.service.GetId(),
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{
						Address:            supplierBech32,
						RevSharePercentage: 100,
					},
				},
			},
		},
		ServicesActivationHeightsMap: map[string]uint64{
			s.service.GetId(): 0,
		},
	}
}

// getProofParams returns the default proof params with a high proof requirement threshold
// and no proof request probability such that no claims require a proof.
func (s *tokenLogicModuleTestSuite) getProofParams() *prooftypes.Params {
	proofParams := prooftypes.DefaultParams()
	highProofRequirementThreshold := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, math.MaxInt64)
	proofParams.ProofRequirementThreshold = &highProofRequirementThreshold
	proofParams.ProofRequestProbability = 0
	return &proofParams
}

// getSharedParams returns the default shared params with the CUTTM set to 1.
func (s *tokenLogicModuleTestSuite) getSharedParams() *sharedtypes.Params {
	sharedParams := sharedtypes.DefaultParams()
	sharedParams.ComputeUnitsToTokensMultiplier = 1
	return &sharedParams
}

// getTokenomicsParams returns the default tokenomics params with the dao_reward_address set to s.daoRewardBech32.
func (s *tokenLogicModuleTestSuite) getTokenomicsParams() *tokenomicstypes.Params {
	tokenomicsParams := tokenomicstypes.DefaultParams()
	tokenomicsParams.DaoRewardAddress = s.daoRewardBech32
	return &tokenomicsParams
}

// createClaim creates numClaims number of claims for the current session given
// the suites service, application, and supplier.
// DEV_NOTE: The sum/count must be large enough to avoid a proposer reward
// (or other small proportion rewards) from being truncated to zero (> 1upokt).
func (s *tokenLogicModuleTestSuite) createClaims(
	keepers *testkeeper.TokenomicsModuleKeepers,
	numClaims int,
) {
	s.T().Helper()

	session, err := s.keepers.GetSession(s.ctx, &sessiontypes.QueryGetSessionRequest{
		ServiceId:          s.service.GetId(),
		ApplicationAddress: s.app.GetAddress(),
		BlockHeight:        1,
	})
	require.NoError(s.T(), err)

	// Create claims (no proof requirements)
	for i := 0; i < numClaims; i++ {
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
func (s *tokenLogicModuleTestSuite) settleClaims(t *testing.T) (settledResults, expiredResults tlm.SettlementResults) {
	// Increment the block height to the settlement height.
	settlementHeight := sharedtypes.GetSettlementSessionEndHeight(s.getSharedParams(), 1)
	s.setBlockHeight(settlementHeight)

	settledPendingResults, expiredPendingResults, err := s.keepers.SettlePendingClaims(cosmostypes.UnwrapSDKContext(s.ctx))
	require.NoError(t, err)

	require.NotZero(t, len(settledPendingResults))
	// TODO_IMPROVE: enhance the test scenario to include expiring claims to increase coverage.
	require.Zero(t, len(expiredPendingResults))

	return settledPendingResults, expiredPendingResults
}

// setBlockHeight sets the block height of the suite's context to height.
func (s *tokenLogicModuleTestSuite) setBlockHeight(height int64) {
	s.ctx = cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(height)
}

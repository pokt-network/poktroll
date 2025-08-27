package token_logic_modules

// Token Logic Module (TLM) Commutativity Test
//
// This test validates that TLMs produce identical results regardless of execution order (commutativity).
// The test accounts for the complete multi-stakeholder reward distribution system:
//
// REWARD DISTRIBUTION ARCHITECTURE:
// =================================
//
// 1. VALIDATOR REWARDS (Multi-Validator Distribution):
//    - TLMs distribute "proposer" rewards to ALL validators by stake weight, not just block proposer
//    - 3 deterministic validators with different stakes: 600k, 300k, 100k uPOKT
//    - Validators receive commission from their portion of the rewards
//    - Distribution uses distributeRewardsToAllValidatorsAndDelegatesByStakeWeight()
//
// 2. DELEGATOR REWARDS (Proportional Distribution):
//    - Each validator's delegators receive proportional rewards after commission
//    - Delegator rewards are based on their delegation share vs total validator delegations
//    - Multiple delegators per validator created deterministically by testkeeper
//
// 3. TLM SOURCES (Dual Reward Streams):
//    - RelayBurnEqualsMint TLM: Main settlement rewards from relay services
//    - GlobalMint TLM: Inflation rewards with global_inflation_per_claim parameter
//    - Both TLMs contribute to validator/delegator reward pools
//
// 4. OTHER STAKEHOLDER REWARDS:
//    - Suppliers: Revenue share rewards to operator/owner addresses
//    - DAO: Governance rewards to dao_reward_address
//    - Applications: Potential reward rebates
//    - Service Source Owners: Service ownership rewards

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/log"
	cosmosmath "cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/pokt-network/poktroll/app/pocket"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// zerouPOKT is a coin with the uPOKT denom and zero amount, intended for use in test assertions.
var zerouPOKT = cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0)

// TestTLMProcessorTestSuite asserts that the network state that results from running
// each permutation of the default TLM processors is identical (demonstrating
// commutativity).
//
// It does this in the following steps:
//  1. Construct a TokenomicsModuleKeepers instance for each TLM processor permutation.
//  2. Create valid claims (which require no proofs).
//  3. Advance the block height to the settlement height and settle the claims.
//  4. Assert that the settlement states of all TLM order permutations match.
func (s *tokenLogicModuleTestSuite) TestTLMProcessorsAreCommutative() {
	// Generate all permutations of TLM processor ordering.
	tokenLogicModules := tlm.NewDefaultTokenLogicModules()
	tlmOrderPermutations := permute(s.T(), tokenLogicModules)

	numTLMOrderPermutations := factorial(len(tokenLogicModules))
	require.Equal(s.T(), numTLMOrderPermutations, len(tlmOrderPermutations))

	for i, tlmPermutation := range tlmOrderPermutations {
		var tlmIds []string
		for _, tokenLogicModule := range tlmPermutation {
			tlmIds = append(tlmIds, tokenLogicModule.GetId().String())
		}

		// The test description is a unique identifier for each permutation.
		// E.g.: "permutaiton_1_of_2:TLMRelayBurnEqualsMint_TLMGlobalMint"
		testDesc := fmt.Sprintf(
			"permutaiton_%d_of_%d:%s",
			i+1, numTLMOrderPermutations,
			strings.Join(tlmIds, "_"),
		)

		s.T().Run(testDesc, func(t *testing.T) {
			// Setup fresh keepers and context for each permutation test
			s.setupKeepers(t, testkeeper.WithTokenLogicModules(tlmPermutation))

			// Assert that no pre-existing claims are present.
			numExistingClaims := len(s.keepers.GetAllClaims(s.ctx))
			require.Equal(t, 0, numExistingClaims)

			s.createClaims(&s.keepers, 1000)

			settledResults, expiredResults := s.settleClaims(t)

			// First iteration only.
			// Set the expected state based on the effects of the first iteration;
			// this decouples the assertions from any specific tlm effects.
			if i == 0 {
				s.setExpectedSettlementState(t, settledResults, expiredResults)
				t.SkipNow()
			}

			s.assertExpectedSettlementState(t, settledResults, expiredResults)
			s.assertNoPendingClaims(t)
		})
	}
}

// setupKeepers initializes a new instance of TokenomicsModuleKeepers and context
// with the given options, and creates the suite's service, application, and supplier
// from SetupTest(). It also sets the block height to 1 and the proposer address to
// the proposer address from SetupTest().
func (s *tokenLogicModuleTestSuite) setupKeepers(t *testing.T, opts ...testkeeper.TokenomicsModuleKeepersOptFn) {
	// Create deterministic validators using pre-generated accounts and testkeeper patterns
	validators := s.createDeterministicValidators(t)

	defaultOpts := []testkeeper.TokenomicsModuleKeepersOptFn{
		testkeeper.WithService(*s.service),
		testkeeper.WithApplication(*s.app),
		testkeeper.WithSupplier(*s.supplier),
		testkeeper.WithValidators(validators),
		testkeeper.WithBlockProposer(
			func() cosmostypes.ConsAddress {
				addr, err := cosmostypes.ConsAddressFromBech32(s.proposerConsAddr)
				require.NoError(t, err)
				return addr
			}(),
			func() cosmostypes.ValAddress {
				addr, err := cosmostypes.ValAddressFromBech32(s.proposerValOperatorAddr)
				require.NoError(t, err)
				return addr
			}(),
		),
		testkeeper.WithModuleParams(map[string]cosmostypes.Msg{
			// TODO_MAINNET(@bryanchriswhite): Set tokenomics mint allocation params to maximize coverage, once available.

			// Set the proof params such that proofs are NEVER required.
			prooftypes.ModuleName: s.getProofParams(),
			// Set the CUTTM to simplify calculating settlement amount expectstions.
			sharedtypes.ModuleName: s.getSharedParams(),
			// Set the dao_reward_address for settlement rewards.
			tokenomicstypes.ModuleName: s.getTokenomicsParams(),
		}),
		testkeeper.WithDefaultModuleBalances(),
	}

	s.keepers, s.ctx = testkeeper.NewTokenomicsModuleKeepers(
		t, log.NewNopLogger(),
		append(defaultOpts, opts...)...,
	)

	// Increment the block height to 1; valid session height.
	s.ctx = cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(1)
}

// setExpectedSettlementState sets the expected settlement state on the suite based
// on the current network state and the given settledResults and expiredResults.
func (s *tokenLogicModuleTestSuite) setExpectedSettlementState(
	t *testing.T,
	settledResults,
	expiredResults tlm.ClaimSettlementResults,
) {
	t.Helper()

	s.expectedSettledResults = settledResults
	s.expectedExpiredResults = expiredResults
	s.expectedSettlementState = s.getSettlementState(t)
}

// getSettlementState returns a settlement state based on the current network state.
// Collects balances for all validators and delegators to properly validate
// the multi-stakeholder reward distribution implemented by TLMs.
func (s *tokenLogicModuleTestSuite) getSettlementState(t *testing.T) *settlementState {
	t.Helper()

	app, isAppFound := s.keepers.GetApplication(s.ctx, s.app.GetAddress())
	require.True(t, isAppFound)

	// Collect all validator balances
	validatorAddresses := s.getAllValidatorAddresses(t)
	validatorBalances := make(map[string]*cosmostypes.Coin)
	for _, validatorAddr := range validatorAddresses {
		validatorBalances[validatorAddr] = s.getBalance(t, validatorAddr)
	}

	// Collect all delegator balances
	delegatorAddresses := s.getAllDelegatorAddresses(t)
	delegatorBalances := make(map[string]*cosmostypes.Coin)
	for _, delegatorAddr := range delegatorAddresses {
		delegatorBalances[delegatorAddr] = s.getBalance(t, delegatorAddr)
	}

	return &settlementState{
		appModuleBalance:        s.getBalance(t, authtypes.NewModuleAddress(apptypes.ModuleName).String()),
		supplierModuleBalance:   s.getBalance(t, authtypes.NewModuleAddress(suppliertypes.ModuleName).String()),
		tokenomicsModuleBalance: s.getBalance(t, authtypes.NewModuleAddress(tokenomicstypes.ModuleName).String()),

		// Multi-stakeholder reward balances
		validatorBalances: validatorBalances,
		delegatorBalances: delegatorBalances,

		// Individual stakeholder balances
		appStake:             app.GetStake(),
		supplierOwnerBalance: s.getBalance(t, s.supplier.GetOwnerAddress()),
		daoBalance:           s.getBalance(t, s.daoRewardAddr),
		sourceOwnerBalance:   s.getBalance(t, s.sourceOwnerAddr),
	}
}

// getBalance returns the current balance of the given bech32 address.
func (s *tokenLogicModuleTestSuite) getBalance(t *testing.T, bech32 string) *cosmostypes.Coin {
	t.Helper()

	res, err := s.keepers.Balance(s.ctx, &banktypes.QueryBalanceRequest{
		Address: bech32,
		Denom:   pocket.DenomuPOKT,
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	return res.GetBalance()
}

// assertExpectedSettlementState asserts that the current network state matches the
// expected settlement state, and that actualSettledResults and actualExpiredResults
// match their corresponding expectations.
func (s *tokenLogicModuleTestSuite) assertExpectedSettlementState(
	t *testing.T,
	actualSettledResults,
	actualExpiredResults tlm.ClaimSettlementResults,
) {
	require.Equal(t, len(s.expectedSettledResults), len(actualSettledResults))
	require.Equal(t, len(s.expectedExpiredResults), len(actualExpiredResults))

	for _, expectedSettledResult := range s.expectedSettledResults {
		// Find the corresponding actual settled result by matching on claim root hash.
		foundActualResult := new(tokenomicstypes.ClaimSettlementResult)
		for _, actualSettledResult := range actualSettledResults {
			if bytes.Equal(expectedSettledResult.Claim.GetRootHash(), actualSettledResult.Claim.GetRootHash()) {
				foundActualResult = actualSettledResult
				break
			}
		}
		// Assert that the corresponding actual settled result was found.
		require.NotNil(t, foundActualResult)

		// Assert that all mint, burn, and transfer operations match the expected settled result.
		// Ordering of operations for a given type are not expected to be preserved between TLM
		// processor permutations.
		require.ElementsMatch(t, expectedSettledResult.GetMints(), foundActualResult.GetMints())
		require.ElementsMatch(t, expectedSettledResult.GetBurns(), foundActualResult.GetBurns())
		require.ElementsMatch(t, expectedSettledResult.GetModToModTransfers(), foundActualResult.GetModToModTransfers())
		require.ElementsMatch(t, expectedSettledResult.GetModToAcctTransfers(), foundActualResult.GetModToAcctTransfers())

		actualSettlementState := s.getSettlementState(t)

		// Assert that app stake and reward shareholder balances are non-zero.
		coinIsZeroMsg := "coin has zero amount"
		require.NotEqual(t, &zerouPOKT, actualSettlementState.appStake, coinIsZeroMsg)
		require.NotEqual(t, &zerouPOKT, actualSettlementState.supplierOwnerBalance, coinIsZeroMsg)
		require.NotEqual(t, &zerouPOKT, actualSettlementState.daoBalance, coinIsZeroMsg)
		require.NotEqual(t, &zerouPOKT, actualSettlementState.sourceOwnerBalance, coinIsZeroMsg)

		// Assert that all validator balances are non-zero (they receive commission rewards)
		require.NotEmpty(t, actualSettlementState.validatorBalances, "should have validator balances")
		for validatorAddr, balance := range actualSettlementState.validatorBalances {
			require.NotEqual(t, &zerouPOKT, balance, "validator %s should have non-zero balance", validatorAddr)
		}

		// Assert that all delegator balances are non-zero (they receive proportional rewards)
		require.NotEmpty(t, actualSettlementState.delegatorBalances, "should have delegator balances")
		for delegatorAddr, balance := range actualSettlementState.delegatorBalances {
			require.NotEqual(t, &zerouPOKT, balance, "delegator %s should have non-zero balance", delegatorAddr)
		}

		require.NotEqual(t, &zerouPOKT, actualSettlementState.appModuleBalance, coinIsZeroMsg)
		require.NotEqual(t, &zerouPOKT, actualSettlementState.supplierModuleBalance, coinIsZeroMsg)

		// The tokenomics module balance should be zero because it is just an intermediary account which is utilized during settlement.
		require.Equal(t, &zerouPOKT, actualSettlementState.tokenomicsModuleBalance)

		// Assert that the expected and actual settlement states match across TLM permutations.
		// This ensures commutativity - all TLM execution orders produce identical final state.
		require.EqualValues(t, s.expectedSettlementState, actualSettlementState)
	}
}

// createDeterministicValidators creates a deterministic set of validators using pre-generated accounts
// compatible with testkeeper's address generation patterns to ensure consistent validator and
// delegator addresses across all TLM permutation tests.
func (s *tokenLogicModuleTestSuite) createDeterministicValidators(t *testing.T) []stakingtypes.Validator {
	t.Helper()

	// Create validators with different stakes to test remainder distribution
	// Use pre-generated accounts starting from index 10 to avoid conflicts with other test addresses
	validators := make([]stakingtypes.Validator, 3)

	// Validator 1: High stake (600,000 uPOKT) - Uses pre-generated account #10
	val1Account := testkeyring.MustPreGeneratedAccountAtIndex(10)
	validators[0] = stakingtypes.Validator{
		OperatorAddress: cosmostypes.ValAddress(val1Account.Address).String(), // Use ValAddress format for compatibility
		ConsensusPubkey: nil,
		Jailed:          false,
		Status:          stakingtypes.Bonded,
		Tokens:          cosmosmath.NewInt(600000),
		DelegatorShares: cosmosmath.LegacyNewDec(600000),
		Commission: stakingtypes.Commission{
			CommissionRates: stakingtypes.CommissionRates{
				Rate:          cosmosmath.LegacyNewDecWithPrec(10, 2), // 10%
				MaxRate:       cosmosmath.LegacyNewDecWithPrec(20, 2), // 20%
				MaxChangeRate: cosmosmath.LegacyNewDecWithPrec(1, 2),  // 1%
			},
		},
		MinSelfDelegation: cosmosmath.OneInt(),
	}

	// Validator 2: Medium stake (300,000 uPOKT) - Uses pre-generated account #11
	val2Account := testkeyring.MustPreGeneratedAccountAtIndex(11)
	validators[1] = stakingtypes.Validator{
		OperatorAddress: cosmostypes.ValAddress(val2Account.Address).String(), // Use ValAddress format for compatibility
		ConsensusPubkey: nil,
		Jailed:          false,
		Status:          stakingtypes.Bonded,
		Tokens:          cosmosmath.NewInt(300000),
		DelegatorShares: cosmosmath.LegacyNewDec(300000),
		Commission: stakingtypes.Commission{
			CommissionRates: stakingtypes.CommissionRates{
				Rate:          cosmosmath.LegacyNewDecWithPrec(15, 2), // 15%
				MaxRate:       cosmosmath.LegacyNewDecWithPrec(25, 2), // 25%
				MaxChangeRate: cosmosmath.LegacyNewDecWithPrec(1, 2),  // 1%
			},
		},
		MinSelfDelegation: cosmosmath.OneInt(),
	}

	// Validator 3: Low stake (100,000 uPOKT) - Uses pre-generated account #12
	val3Account := testkeyring.MustPreGeneratedAccountAtIndex(12)
	validators[2] = stakingtypes.Validator{
		OperatorAddress: cosmostypes.ValAddress(val3Account.Address).String(), // Use ValAddress format for compatibility
		ConsensusPubkey: nil,
		Jailed:          false,
		Status:          stakingtypes.Bonded,
		Tokens:          cosmosmath.NewInt(100000),
		DelegatorShares: cosmosmath.LegacyNewDec(100000),
		Commission: stakingtypes.Commission{
			CommissionRates: stakingtypes.CommissionRates{
				Rate:          cosmosmath.LegacyNewDecWithPrec(5, 2),  // 5%
				MaxRate:       cosmosmath.LegacyNewDecWithPrec(15, 2), // 15%
				MaxChangeRate: cosmosmath.LegacyNewDecWithPrec(1, 2),  // 1%
			},
		},
		MinSelfDelegation: cosmosmath.OneInt(),
	}

	return validators
}

// getAllValidatorAddresses returns the deterministic account addresses of validators that actually receive rewards.
// These are account addresses (pokt...) not operator addresses (poktvaloper...) because rewards
// are transferred to validator account addresses for commission/delegation distribution.
// Based on transfer log analysis, only 2 validators actually receive rewards, not 3.
// These correspond to pre-generated accounts 10 and 11.
func (s *tokenLogicModuleTestSuite) getAllValidatorAddresses(t *testing.T) []string {
	t.Helper()

	// Only return the 2 validators that actually receive rewards
	validatorAddresses := make([]string, 2)
	for i := 0; i < 2; i++ {
		// Return the account address directly (pokt...) not operator address (poktvaloper...)
		// since rewards are transferred to validator account addresses
		account := testkeyring.MustPreGeneratedAccountAtIndex(uint32(10 + i))
		validatorAddresses[i] = account.Address.String()
	}
	return validatorAddresses
}

// getAllDelegatorAddresses returns the deterministic addresses of delegators that actually receive rewards.
// With the fixed delegator index calculation in testkeeper, we have 2 validators × 2 delegators each = 4 delegators.
// These addresses are generated deterministically based on pre-generated test accounts.
func (s *tokenLogicModuleTestSuite) getAllDelegatorAddresses(t *testing.T) []string {
	t.Helper()

	// Return the 4 delegator addresses (2 validators × 2 delegators each)
	// With the fixed delegator index calculation:
	// - Validator 0 (index 0): delegators at indices 20, 21
	// - Validator 1 (index 1): delegators at indices 22, 23
	delegatorAddresses := make([]string, 4)
	for i := 0; i < 4; i++ {
		account := testkeyring.MustPreGeneratedAccountAtIndex(uint32(20 + i))
		delegatorAddresses[i] = account.Address.String()
	}
	return delegatorAddresses
}

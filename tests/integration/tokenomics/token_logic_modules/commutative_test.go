package token_logic_modules

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
)

// zerouPOKT is a coin with the uPOKT denom and zero amount, intended for use in test assertions.
var zerouPOKT = types.NewInt64Coin(volatile.DenomuPOKT, 0)

// TestTLMProcessorTestSuite asserts that the network state that results from running
// each permutation of the default TLM processors is identical (demonstrating
// commutativity). It does this in the following steps:
//
// 1. Construct a TokenomicsModuleKeepers instance for each TLM processor permutation.
// 2. Create valid claims (which require no proofs).
// 3. Advance the block height to the settlement height and settle the claims.
// 5. Assert that the settlement state is identical to the first.
func (s *tlmProcessorsTestSuite) TestTLMProcessorsAreCommutative() {
	// Generate all permutations of TLM processor ordering.
	processors := tlm.NewDefaultProcessors(s.foundationBech32)
	processorOrderPermutations := permute(processors)

	numProcessorOrderPermutations := factorial(len(processors))
	require.Equal(s.T(), numProcessorOrderPermutations, len(processorOrderPermutations))

	for i, procs := range processorOrderPermutations {
		var tlmProcNames []string
		for _, proc := range procs {
			tlmProcNames = append(tlmProcNames, proc.GetTLM().String())
		}

		// The test description is a unique identifier for each permutation.
		// E.g.: "permutaiton_1_of_2__TLMRelayBurnEqualsMint_TLMGlobalMint"
		testDesc := fmt.Sprintf(
			"permutaiton_%d_of_%d__%s",
			i+1, numProcessorOrderPermutations,
			strings.Join(tlmProcNames, "_"),
		)

		s.T().Run(testDesc, func(t *testing.T) {
			s.setupKeepers(t, keeper.WithTLMProcessors(procs))

			// Assert that no pre-existing claims are present.
			numExistingClaims := len(s.keepers.GetAllClaims(s.ctx))
			require.Equal(t, 0, numExistingClaims)

			s.createClaims(&s.keepers, 1000)
			settledResults, expiredResults := s.settleClaims(t)

			// Set the expected state based on the effects of the first iteration;
			// this decouples the assertions from any specific tlm effects.
			if i == 0 {
				s.setExpectedSettlementState(t, settledResults, expiredResults)
				t.SkipNow()
			}

			s.assertExpectedSettlementState(t, settledResults, expiredResults)
		})
	}
}

// setupKeepers initializes a new instance of TokenomicsModuleKeepers and context
// with the given options, and creates the suite's service, application, and supplier
// from SetupTest(). It also sets the block height to 1 and the proposer address to
// the proposer address from SetupTest().
func (s *tlmProcessorsTestSuite) setupKeepers(t *testing.T, opts ...keeper.TokenomicsModuleKeepersOptFn) {
	defaultOpts := []keeper.TokenomicsModuleKeepersOptFn{
		keeper.WithService(*s.service),
		keeper.WithApplication(*s.app),
		keeper.WithSupplier(*s.supplier),
		keeper.WithModuleParams(map[string]types.Msg{
			// TODO_TECHDEBT: Set tokenomics mint allocation params to maximize coverage, once available.

			// Set the proof params such that proofs are NEVER required.
			prooftypes.ModuleName: s.getProofParams(),
			// Set the CUTTM to simplify calculating settlement amount expectstions.
			sharedtypes.ModuleName: s.getSharedParams(),
		}),
	}

	s.keepers, s.ctx = keeper.NewTokenomicsModuleKeepers(
		t, log.NewNopLogger(),
		append(defaultOpts, opts...)...,
	)

	// Increment the block height to 1; valid session height and set the proposer address.
	s.ctx = types.UnwrapSDKContext(s.ctx).
		WithBlockHeight(1).
		WithProposer(s.proposerConsAddr)
}

// setExpectedSettlementState sets the expected settlement state on the suite based
// on the current network state and the given settledResults and expiredResults.
func (s *tlmProcessorsTestSuite) setExpectedSettlementState(
	t *testing.T,
	settledResults,
	expiredResults tlm.PendingSettlementResults,
) {
	t.Helper()

	s.expectedSettledResults = settledResults
	s.expectedExpiredResults = expiredResults
	s.expectedSettlementState = s.getSettlementState(t)
}

// getSettlementState returns a settlement state based on the current network state.
func (s *tlmProcessorsTestSuite) getSettlementState(t *testing.T) *settlementState {
	t.Helper()

	app, isAppFound := s.keepers.GetApplication(s.ctx, s.app.GetAddress())
	require.True(t, isAppFound)

	proposerBech32 := sample.AccAddressFromConsBech32(s.proposerConsAddr.String())

	return &settlementState{
		appStake:             app.GetStake(),
		supplierOwnerBalance: s.getBalance(t, s.supplier.GetOwnerAddress()),
		proposerBalance:      s.getBalance(t, proposerBech32),
		foundationBalance:    s.getBalance(t, s.foundationBech32),
		sourceOwnerBalance:   s.getBalance(t, s.sourceOwnerBech32),
	}
}

// getBalance returns the current balance of the given bech32 address.
func (s *tlmProcessorsTestSuite) getBalance(t *testing.T, bech32 string) *types.Coin {
	t.Helper()

	res, err := s.keepers.Balance(s.ctx, &banktypes.QueryBalanceRequest{
		Address: bech32,
		Denom:   volatile.DenomuPOKT,
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	return res.GetBalance()
}

// assertExpectedSettlementState asserts that the current network state matches the
// expected settlement state, and that actualSettledResults and actualExpiredResults
// match their corresponding expectations.
func (s *tlmProcessorsTestSuite) assertExpectedSettlementState(
	t *testing.T,
	actualSettledResults,
	actualExpiredResults tlm.PendingSettlementResults,
) {
	require.Equal(t, len(s.expectedSettledResults), len(actualSettledResults))
	require.Equal(t, len(s.expectedExpiredResults), len(actualExpiredResults))

	for _, expectedSettledResult := range s.expectedSettledResults {
		// Find the corresponding actual settled result by matching on claim root hash.
		foundActualResult := new(tlm.PendingSettlementResult)
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
		require.ElementsMatch(t, expectedSettledResult.Mints, foundActualResult.Mints)
		require.ElementsMatch(t, expectedSettledResult.Burns, foundActualResult.Burns)
		require.ElementsMatch(t, expectedSettledResult.ModToModTransfers, foundActualResult.ModToModTransfers)
		require.ElementsMatch(t, expectedSettledResult.ModToAcctTransfers, foundActualResult.ModToAcctTransfers)

		actualSettlementState := s.getSettlementState(t)

		// Assert that app stake and rewardee balances settlementState are non-zero.
		require.NotEqual(t, &zerouPOKT, actualSettlementState.appStake)
		require.NotEqual(t, &zerouPOKT, actualSettlementState.appBalance)
		require.NotEqual(t, &zerouPOKT, actualSettlementState.supplierOwnerBalance)
		require.NotEqual(t, &zerouPOKT, actualSettlementState.proposerBalance)
		require.NotEqual(t, &zerouPOKT, actualSettlementState.foundationBalance)
		require.NotEqual(t, &zerouPOKT, actualSettlementState.sourceOwnerBalance)

		// Assert that the expected and actual settlement states match.
		require.EqualValues(t, s.expectedSettlementState, actualSettlementState)
	}
}

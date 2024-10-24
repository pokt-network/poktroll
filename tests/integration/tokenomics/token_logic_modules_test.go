package integration

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"
	"testing"

	cosmoslog "cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
)

var zerouPOKT = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)

type tlmProcessorsTestSuite struct {
	suite.Suite

	ctx     context.Context
	keepers testkeeper.TokenomicsModuleKeepers

	service  *sharedtypes.Service
	app      *apptypes.Application
	supplier *sharedtypes.Supplier

	proposerConsAddr cosmostypes.ConsAddress
	sourceOwnerBech32,
	foundationBech32 string

	// TODO_IN_THIS_COMMIT: godoc...
	expectedSettledResults,
	expectedExpiredResults tlm.PendingSettlementResults
	expectedSettlementState *expectedSettlementState
}

// TODO_IN_THIS_COMMIT: godoc... field names MUST be exported to assert via reflection...
type expectedSettlementState struct {
	appStake             *cosmostypes.Coin
	appBalance           *cosmostypes.Coin
	supplierOwnerBalance *cosmostypes.Coin
	proposerBalance      *cosmostypes.Coin
	foundationBalance    *cosmostypes.Coin
	sourceOwnerBalance   *cosmostypes.Coin
}

func TestTLMProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(tlmProcessorsTestSuite))
}

func (s *tlmProcessorsTestSuite) SetupTest() {
	s.foundationBech32 = sample.AccAddress()
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
						RevSharePercentage: 50,
					},
				},
			},
		},
		ServicesActivationHeightsMap: map[string]uint64{
			s.service.GetId(): 0,
		},
	}
}

func (s *tlmProcessorsTestSuite) setupKeepers(t *testing.T, opts ...testkeeper.TokenomicsModuleKeepersOpt) {
	defaultOpts := []testkeeper.TokenomicsModuleKeepersOpt{
		testkeeper.WithService(*s.service),
		testkeeper.WithApplication(*s.app),
		testkeeper.WithSupplier(*s.supplier),
		testkeeper.WithModuleParams(map[string]cosmostypes.Msg{
			// TODO_TECHDEBT: Set tokenomics mint allocation params to maximize coverage, once available.

			// Set the proof params such that proofs are NEVER required.
			prooftypes.ModuleName: s.getProofParams(),
			// Set the CUTTM to simplify calculating settlement amount expectstions.
			sharedtypes.ModuleName: s.getSharedParams(),
		}),
	}

	s.keepers, s.ctx = testkeeper.NewTokenomicsModuleKeepers(
		t, cosmoslog.NewNopLogger(),
		append(defaultOpts, opts...)...,
	)

	// Increment the block height to 1; valid session height and set the proposer address.
	s.ctx = cosmostypes.UnwrapSDKContext(s.ctx).
		WithBlockHeight(1).
		WithProposer(s.proposerConsAddr)
}

func (s *tlmProcessorsTestSuite) TestTLMProcessorsAreCommutative() {
	// Generate all permutations of TLM processor ordering.
	processors := tlm.NewDefaultProcessors(s.foundationBech32)
	processorOrderPermutations := permute(processors)

	numProcessorOrderPermutations := factorial(len(processors))
	require.Equal(s.T(), numProcessorOrderPermutations, len(processorOrderPermutations))

	// TODO_IN_THIS_COMMIT: update comment...
	// Apply each permutation of TLM processors to a test case.
	// Assert that ALL results are identical.
	for i, procs := range processorOrderPermutations {
		var tlmProcNames []string
		for _, proc := range procs {
			tlmProcNames = append(tlmProcNames, proc.GetTLM().String())
		}

		testDesc := fmt.Sprintf("permutaiton_%d__%s", i, strings.Join(tlmProcNames, "_"))
		s.T().Run(testDesc, func(t *testing.T) {
			s.setupKeepers(t, testkeeper.WithTLMProcessors(procs))

			// Assert that no pre-existing claims are present.
			numExistingClaims := len(s.keepers.GetAllClaims(s.ctx))
			require.Equal(t, 0, numExistingClaims)

			s.createClaims(&s.keepers, 1000)
			settledResults, expiredResults := s.settleClaims(t)

			// TODO_IN_THIS_COMMIT: comment... set the expected state on the first iteration... independent of specific tlm effects...
			if i == 0 {
				s.setExpectedSettlementState(t, settledResults, expiredResults)
				t.SkipNow()
			}

			s.assertExpectedSettlementState(t, settledResults, expiredResults)
		})
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
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

// TODO_IN_THIS_COMMIT: move & godoc...
func (s *tlmProcessorsTestSuite) getSettlementState(t *testing.T) *expectedSettlementState {
	t.Helper()

	app, isAppFound := s.keepers.GetApplication(s.ctx, s.app.GetAddress())
	require.True(t, isAppFound)

	proposerBech32 := sample.AccAddressFromConsBech32(s.proposerConsAddr.String())

	return &expectedSettlementState{
		appStake:             app.GetStake(),
		supplierOwnerBalance: s.getBalance(t, s.supplier.GetOwnerAddress()),
		proposerBalance:      s.getBalance(t, proposerBech32),
		foundationBalance:    s.getBalance(t, s.foundationBech32),
		sourceOwnerBalance:   s.getBalance(t, s.sourceOwnerBech32),
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (s *tlmProcessorsTestSuite) getBalance(t *testing.T, bech32 string) *cosmostypes.Coin {
	t.Helper()

	res, err := s.keepers.Balance(s.ctx, &banktypes.QueryBalanceRequest{
		Address: bech32,
		Denom:   volatile.DenomuPOKT,
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	return res.GetBalance()
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (s *tlmProcessorsTestSuite) assertExpectedSettlementState(
	t *testing.T,
	actualSettledResults,
	actualExpiredResults tlm.PendingSettlementResults,
) {
	actualSettlementState := s.getSettlementState(t)

	// Assert that app stake and rewardee balances expectedSettlementState are non-zero.
	require.NotEqual(t, &zerouPOKT, actualSettlementState.appStake)
	require.NotEqual(t, &zerouPOKT, actualSettlementState.appBalance)
	require.NotEqual(t, &zerouPOKT, actualSettlementState.supplierOwnerBalance)
	require.NotEqual(t, &zerouPOKT, actualSettlementState.proposerBalance)
	require.NotEqual(t, &zerouPOKT, actualSettlementState.foundationBalance)
	require.NotEqual(t, &zerouPOKT, actualSettlementState.sourceOwnerBalance)

	require.EqualValues(t, s.expectedSettlementState, actualSettlementState)

	require.Equal(t, len(s.expectedSettledResults), len(actualSettledResults))
	require.Equal(t, len(s.expectedExpiredResults), len(actualExpiredResults))

	for _, expectedSettledResult := range s.expectedSettledResults {
		// Find the corresponding actual settled result.
		foundActualResult := new(tlm.PendingSettlementResult)
		for _, actualSettledResult := range actualSettledResults {
			if bytes.Equal(expectedSettledResult.Claim.GetRootHash(), actualSettledResult.Claim.GetRootHash()) {
				foundActualResult = actualSettledResult
				break
			}
		}
		require.NotNil(t, foundActualResult)

		require.ElementsMatch(t, expectedSettledResult.Mints, foundActualResult.Mints)
		require.ElementsMatch(t, expectedSettledResult.Burns, foundActualResult.Burns)
		require.ElementsMatch(t, expectedSettledResult.ModToModTransfers, foundActualResult.ModToModTransfers)
		require.ElementsMatch(t, expectedSettledResult.ModToAcctTransfers, foundActualResult.ModToAcctTransfers)
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (s *tlmProcessorsTestSuite) getProofParams() *prooftypes.Params {
	proofParams := prooftypes.DefaultParams()
	highProofRequirementThreshold := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, math.MaxInt64)
	proofParams.ProofRequirementThreshold = &highProofRequirementThreshold
	proofParams.ProofRequestProbability = 0
	return &proofParams
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (s *tlmProcessorsTestSuite) getSharedParams() *sharedtypes.Params {
	sharedParams := sharedtypes.DefaultParams()
	sharedParams.ComputeUnitsToTokensMultiplier = 1
	return &sharedParams
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *tlmProcessorsTestSuite) createClaims(
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

func (s *tlmProcessorsTestSuite) settleClaims(t *testing.T) (settledResults, expiredResults tlm.PendingSettlementResults) {
	// Increment the block height to the settlement height.
	settlementHeight := sharedtypes.GetSettlementSessionEndHeight(s.getSharedParams(), 1)
	s.setBlockHeight(settlementHeight)

	settledPendingResults, expiredPendingResults, err := s.keepers.SettlePendingClaims(cosmostypes.UnwrapSDKContext(s.ctx))
	require.NoError(t, err)

	require.NotZero(t, 1, len(settledPendingResults))
	// TODO_IMPROVE: enhance the test scenario to include expiring claims to increase coverage.
	require.Zero(t, len(expiredPendingResults))

	return settledPendingResults, expiredPendingResults
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (s *tlmProcessorsTestSuite) setBlockHeight(height int64) {
	s.ctx = cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(height)
}

// TODO_TEST(@bryanchriswhite): Settlement proceeds in the face of errors
// - Does not block settling of other claims in the same session
// - Does not block setting subsequent sessions

func TestPermute(t *testing.T) {
	input := []string{"1", "2", "3", "4"}
	expected := map[string][]string{
		"1234": {"1", "2", "3", "4"},
		"1243": {"1", "2", "4", "3"},
		"1342": {"1", "3", "4", "2"},
		"1324": {"1", "3", "2", "4"},
		"1423": {"1", "4", "2", "3"},
		"1432": {"1", "4", "3", "2"},
		"2341": {"2", "3", "4", "1"},
		"2314": {"2", "3", "1", "4"},
		"2413": {"2", "4", "1", "3"},
		"2431": {"2", "4", "3", "1"},
		"2143": {"2", "1", "4", "3"},
		"2134": {"2", "1", "3", "4"},
		"3412": {"3", "4", "1", "2"},
		"3421": {"3", "4", "2", "1"},
		"3124": {"3", "1", "2", "4"},
		"3142": {"3", "1", "4", "2"},
		"3241": {"3", "2", "4", "1"},
		"3214": {"3", "2", "1", "4"},
		"4123": {"4", "1", "2", "3"},
		"4132": {"4", "1", "3", "2"},
		"4231": {"4", "2", "3", "1"},
		"4213": {"4", "2", "1", "3"},
		"4312": {"4", "3", "1", "2"},
		"4321": {"4", "3", "2", "1"},
	}

	actual := permute(input)
	require.Equal(t, factorial(len(input)), len(actual))

	// Assert that each actual result matches exactly one expected permutation.
	for _, actualPermutation := range expected {
		actualKey := strings.Join(actualPermutation, "")
		expectedPermutation, isExpectedPermutation := expected[actualKey]
		require.True(t, isExpectedPermutation)
		require.Equal(t, expectedPermutation, actualPermutation)

		// Remove observed expected permutation to identify any
		// missing permutations after the loop.
		delete(expected, actualKey)
	}
	// Assert that all expected permutations were observed (and deleted).
	require.Len(t, expected, 0)
}

// permute generates all possible permutations of the input slice 'items'.
func permute[T any](items []T) [][]T {
	var permutations [][]T
	// Create a copy to avoid modifying the original slice.
	itemsCopy := make([]T, len(items))
	copy(itemsCopy, items)
	// Start the recursive permutation generation with swap index 0.
	recursivePermute(itemsCopy, &permutations, 0)
	return permutations
}

// recursivePermute recursively generates permutations by swapping elements.
func recursivePermute[T any](items []T, permutations *[][]T, swapIdx int) {
	if swapIdx == len(items) {
		// Append a copy of the current permutation to the result.
		permutation := make([]T, len(items))
		copy(permutation, items)
		*permutations = append(*permutations, permutation)
		return
	}
	for i := swapIdx; i < len(items); i++ {
		// Swap the current element with the element at the swap index.
		items[swapIdx], items[i] = items[i], items[swapIdx]
		// Recurse with the next swap index.
		recursivePermute[T](items, permutations, swapIdx+1)
		// Swap back to restore the original state (backtrack).
		items[swapIdx], items[i] = items[i], items[swapIdx]
	}
}

func factorial(n int) int {
	if n < 0 {
		return 0 // Handle negative input as an invalid case
	}
	result := 1
	for i := 2; i <= n; i++ {
		result *= i
	}
	return result
}

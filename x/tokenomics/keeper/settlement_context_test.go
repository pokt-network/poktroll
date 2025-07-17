package keeper_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testtree"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const computeUnitsPerRelay = 1

var (
	// Test settlements with claims from multiple services
	testServiceIds   = []string{"svc1", "svc2", "svc3"}
	supplierStakeAmt = 2 * suppliertypes.DefaultMinStake.Amount.Int64()
)

func init() {
	cmd.InitSDKConfig()
}

// TODO_IMPROVE: Consolidate the setup for all tests that use TokenomicsModuleKeepers
type TestSuite struct {
	suite.Suite

	ctx     context.Context
	keepers keepertest.TokenomicsModuleKeepers
	claims  []prooftypes.Claim
	proofs  []prooftypes.Proof

	numRelays                uint64
	numClaimedComputeUnits   uint64
	numEstimatedComputeUnits uint64
	claimedUpokt             cosmostypes.Coin
	relayMiningDifficulties  []servicetypes.RelayMiningDifficulty
}

// SetupTest creates the following and stores them in the suite:
// - An cosmostypes.Context.
// - A keepertest.TokenomicsModuleKeepers to provide access to integrated keepers.
// - An expectedComputeUnits which is the default proof_requirement_threshold.
// - A claim that will require a proof via threshold, given the default proof params.
// - A proof which contains only the session header supplier operator address.
func (s *TestSuite) SetupTest() {
	t := s.T()

	moduleBalancesOpt := keepertest.WithModuleAccountBalances(map[string]int64{
		apptypes.ModuleName:      1000000000,
		suppliertypes.ModuleName: supplierStakeAmt,
	})
	s.keepers, s.ctx = keepertest.NewTokenomicsModuleKeepers(s.T(), nil, moduleBalancesOpt)
	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(1)

	// Add a block proposer address to the context
	sdkCtx = sdkCtx.WithProposer(sample.ConsAddress())

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(s.keepers.Codec)

	// Construct a ringClient to get the application's ring & verify the relay
	// request signature.
	ringClient, err := rings.NewRingClient(depinject.Supply(
		polyzero.NewLogger(),
		prooftypes.NewAppKeeperQueryClient(s.keepers.ApplicationKeeper),
		prooftypes.NewAccountKeeperQueryClient(s.keepers.AccountKeeper),
		prooftypes.NewSharedKeeperQueryClient(s.keepers.SharedKeeper, s.keepers.SessionKeeper),
	))
	require.NoError(t, err)

	// Create a supplier and applications onchain accounts.
	appAddresses, supplierOwnerAddr := s.createTestActors(t, sdkCtx, keyRing)

	s.claims, s.proofs = s.createTestClaimsAndProofs(
		t, sdkCtx, appAddresses, supplierOwnerAddr, keyRing, ringClient,
	)

}

// TestSettlePendingClaims tests the claim settlement process.
// NB: Each test scenario (method) is run in isolation and #TestSetup() is called
// for each prior to running.
func TestSettlePendingClaims(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestSettlePendingClaims_ClaimPendingBeforeSettlement() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	// Use a single claim for this test
	claim := s.claims[0]

	s.keepers.UpsertClaim(ctx, claim)

	// Settle pending claims while the session is still active.
	// Expectations: No claims should be settled because the session is still ongoing
	blockHeight := claim.SessionHeader.SessionEndBlockHeight - 2 // session is still active
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, expiredResults, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled or expired.
	require.Equal(t, uint64(0), settledResults.GetNumClaims())
	require.Equal(t, uint64(0), expiredResults.GetNumClaims())
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)

	// Validate that one claim still remains.
	claims := s.keepers.GetAllClaims(ctx)
	// DEV_NOTE: Using `require.Equal()` over `require.Len()` so errors are easier to read.
	require.Equal(t, 1, len(claims))

	// Calculate a block height which is within the proof window.
	proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(
		&sharedParams, claim.SessionHeader.SessionEndBlockHeight,
	)
	proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(
		&sharedParams, claim.SessionHeader.SessionEndBlockHeight,
	)
	blockHeight = (proofWindowCloseHeight - proofWindowOpenHeight) / 2

	// 2. Settle pending claims just after the session ended.
	// Expectations: Claims should not be settled because the proof window hasn't closed yet.
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResults, expiredResults, numDiscardedFaultyClaims, err = s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled or expired.
	require.Equal(t, uint64(0), settledResults.GetNumClaims())
	require.Equal(t, uint64(0), expiredResults.GetNumClaims())
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)

	// Validate that the claim still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Equal(t, 1, len(claims))
}

func (s *TestSuite) TestSettlePendingClaims_ClaimExpired_ProofRequiredAndNotProvided_ViaThreshold() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	// Use a single claim for this test
	claim := s.claims[0]
	relayMiningDifficulty := s.relayMiningDifficulties[0]

	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)

	// -1 to push threshold below s.claim's compute units
	proofRequirementThreshold = proofRequirementThreshold.Sub(uPOKTCoin(1))

	// Set the proof missing penalty to half the supplier's stake so it is not
	// unstaked when being slashed.
	belowStakeAmountProofMissingPenalty := cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewInt(supplierStakeAmt/2))

	// Set the proof parameters such that s.claim requires a proof because:
	// - proof_request_probability is 0%
	// - proof_requirement_threshold is below the claim (i.e. claim is above threshold)
	err = s.keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability:   0,
		ProofRequirementThreshold: &proofRequirementThreshold,
		ProofMissingPenalty:       &belowStakeAmountProofMissingPenalty,
	})
	require.NoError(t, err)

	// Upsert the claim ONLY
	s.keepers.UpsertClaim(ctx, claim)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be expired.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, expiredResults, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(0), settledResults.GetNumClaims()) // 0 claims settled
	require.Equal(t, uint64(1), expiredResults.GetNumClaims()) // 1 claim expired
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)      // 0 claim discarded

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Equal(t, 0, len(claims))

	// Slashing should have occurred without unstaking the supplier.
	// The supplier is not unstaked because it got slashed by an amount that is
	// half its stake (i.e. missing proof penalty == stake / 2), resulting in a
	// remaining stake that is above the minimum stake (i.e. new_stake == prev_stake / 2).
	slashedSupplier, supplierFound := s.keepers.GetSupplier(sdkCtx, claim.SupplierOperatorAddress)
	require.True(t, supplierFound)
	require.Equal(t, supplierStakeAmt/2, slashedSupplier.Stake.Amount.Int64())
	require.Equal(t, uint64(0), slashedSupplier.UnstakeSessionEndHeight)

	// Validate the supplier and tokenomics module balances.
	supplierModuleBalRes, err := s.keepers.Balance(s.ctx, &banktypes.QueryBalanceRequest{
		Address: authtypes.NewModuleAddress(suppliertypes.ModuleName).String(),
		Denom:   pocket.DenomuPOKT,
	})
	require.NoError(t, err)
	require.Equal(t, supplierStakeAmt/2, supplierModuleBalRes.Balance.Amount.Int64())

	tokenomicsModuleBalRes, err := s.keepers.Balance(s.ctx, &banktypes.QueryBalanceRequest{
		Address: authtypes.NewModuleAddress(tokenomicstypes.ModuleName).String(),
		Denom:   pocket.DenomuPOKT,
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), tokenomicsModuleBalRes.Balance.Amount.Int64())

	events := sdkCtx.EventManager().Events()
	require.Equal(t, 12, len(events)) // asserting on the length of events so the developer must consciously update it upon changes

	// Confirm an expiration event was emitted
	expectedClaimExpiredEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimExpired](t, events)
	require.Equal(t, 1, len(expectedClaimExpiredEvents))

	// Validate the claim expired event
	expectedClaimExpiredEvent := expectedClaimExpiredEvents[0]
	require.Equal(t, tokenomicstypes.ClaimExpirationReason_PROOF_MISSING, expectedClaimExpiredEvent.GetExpirationReason())
	require.Equal(t, s.numRelays, expectedClaimExpiredEvent.GetNumRelays())
	require.Equal(t, s.numClaimedComputeUnits, expectedClaimExpiredEvent.GetNumClaimedComputeUnits())
	require.Equal(t, s.numEstimatedComputeUnits, expectedClaimExpiredEvent.GetNumEstimatedComputeUnits())
	require.Equal(t, s.claimedUpokt.String(), expectedClaimExpiredEvent.GetClaimedUpokt())

	// Confirm that a slashing event was emitted
	expectedSlashingEvents := testutilevents.FilterEvents[*tokenomicstypes.EventSupplierSlashed](t, events)
	require.Equal(t, 1, len(expectedSlashingEvents))

	// Validate the slashing event
	expectedSlashingEvent := expectedSlashingEvents[0]

	require.Equal(t, slashedSupplier.GetOperatorAddress(), expectedSlashingEvent.GetClaim().GetSupplierOperatorAddress())
	require.Equal(t, belowStakeAmountProofMissingPenalty.String(), expectedSlashingEvent.GetProofMissingPenalty())
}

func (s *TestSuite) TestSettlePendingClaims_ClaimSettled_ProofRequiredAndProvided_ViaThreshold() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	// Use a single claim and proof for this test
	claim := s.claims[0]
	proof := s.proofs[0]
	relayMiningDifficulty := s.relayMiningDifficulties[0]

	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)

	// -1 to push threshold below s.claim's compute units
	proofRequirementThreshold = proofRequirementThreshold.Sub(uPOKTCoin(1))

	// Set the proof parameters such that s.claim requires a proof because:
	// - proof_request_probability is 0%
	// - proof_requirement_threshold is below the claim (i.e. claim is above threshold)
	err = s.keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability:   0,
		ProofRequirementThreshold: &proofRequirementThreshold,
	})
	require.NoError(t, err)

	// Upsert the claim & proof
	s.keepers.UpsertClaim(ctx, claim)
	s.keepers.UpsertProof(ctx, proof)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	s.keepers.ValidateSubmittedProofs(sdkCtx)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResult, expiredResult, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(1), settledResult.GetNumClaims()) // 1 claim settled
	require.Equal(t, uint64(0), expiredResult.GetNumClaims()) // 0 claims expired
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)     // 0 claims discarded

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Equal(t, 0, len(claims))

	// Confirm an settlement event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events)
	require.Equal(t, 1, len(expectedEvents))

	// Validate the event
	expectedEvent := expectedEvents[0]
	require.Equal(t, prooftypes.ProofRequirementReason_THRESHOLD, expectedEvent.GetProofRequirement())
	require.Equal(t, s.numRelays, expectedEvent.GetNumRelays())
	require.Equal(t, s.numClaimedComputeUnits, expectedEvent.GetNumClaimedComputeUnits())
	require.Equal(t, s.numEstimatedComputeUnits, expectedEvent.GetNumEstimatedComputeUnits())
	require.Equal(t, s.claimedUpokt.String(), expectedEvent.GetClaimedUpokt())
}

func (s *TestSuite) TestSettlePendingClaims_ClaimExpired_ProofRequired_InvalidOneProvided() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Use a single claim and proof for this test
	claim := s.claims[0]
	proof := s.proofs[0]

	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	// Set the proof parameters such that s.claim DOES NOT require a proof because:
	// - proof_request_probability is 100%
	proofParams.ProofRequestProbability = 1
	// Set the proof missing penalty to half the supplier's stake so it is not
	// unstaked when being slashed.
	belowStakeAmountProofMissingPenalty := cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewInt(supplierStakeAmt/2))
	proofParams.ProofMissingPenalty = &belowStakeAmountProofMissingPenalty
	err := s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Create a claim that requires a proof and an invalid proof
	proof.ClosestMerkleProof = []byte("invalid_proof")

	// Upsert the proof & claim
	s.keepers.UpsertClaim(ctx, claim)
	s.keepers.UpsertProof(ctx, proof)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	s.keepers.ValidateSubmittedProofs(sdkCtx)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be expired.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResults, expiredResults, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(0), settledResults.GetNumClaims()) // 0 claims settled
	require.Equal(t, uint64(1), expiredResults.GetNumClaims()) // 1 claim expired
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)      // 0 claims discarded

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Equal(t, 0, len(claims))

	// Validate that no proofs remain.
	proofs := s.keepers.GetAllProofs(ctx)
	require.Equal(t, 0, len(proofs))

	// Slashing should have occurred without unstaking the supplier.
	slashedSupplier, supplierFound := s.keepers.GetSupplier(sdkCtx, claim.SupplierOperatorAddress)
	require.True(t, supplierFound)
	require.Equal(t, math.NewInt(supplierStakeAmt/2), slashedSupplier.Stake.Amount)
	require.Equal(t, uint64(0), slashedSupplier.UnstakeSessionEndHeight)

	// Confirm an expiration event was emitted
	events := sdkCtx.EventManager().Events()
	require.Equal(t, 13, len(events)) // minting, burning, settling, etc..

	expectedClaimExpiredEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimExpired](t, events)
	require.Equal(t, 1, len(expectedClaimExpiredEvents))

	// Confirm an invalid proof removed event was emitted
	expectedProofValidityCheckedEvents := testutilevents.FilterEvents[*prooftypes.EventProofValidityChecked](t, events)
	require.Equal(t, 1, len(expectedProofValidityCheckedEvents))

	// Validate the event
	expectedClaimExpiredEvent := expectedClaimExpiredEvents[0]
	require.Equal(t, tokenomicstypes.ClaimExpirationReason_PROOF_INVALID, expectedClaimExpiredEvent.GetExpirationReason())
	require.Equal(t, s.numRelays, expectedClaimExpiredEvent.GetNumRelays())
	require.Equal(t, s.numClaimedComputeUnits, expectedClaimExpiredEvent.GetNumClaimedComputeUnits())
	require.Equal(t, s.numEstimatedComputeUnits, expectedClaimExpiredEvent.GetNumEstimatedComputeUnits())
	require.Equal(t, s.claimedUpokt.String(), expectedClaimExpiredEvent.GetClaimedUpokt())

	expectedProofValidityCheckedEvent := expectedProofValidityCheckedEvents[0]
	require.Equal(t, claim.SessionHeader.SessionId, expectedProofValidityCheckedEvent.GetSessionId())
	require.Equal(t, claim.SupplierOperatorAddress, expectedProofValidityCheckedEvent.GetSupplierOperatorAddress())

	// Confirm that a slashing event was emitted
	expectedSlashingEvents := testutilevents.FilterEvents[*tokenomicstypes.EventSupplierSlashed](t, events)
	require.Equal(t, 1, len(expectedSlashingEvents))

	// Validate the slashing event
	expectedSlashingEvent := expectedSlashingEvents[0]
	require.Equal(t, slashedSupplier.GetOperatorAddress(), expectedSlashingEvent.GetClaim().GetSupplierOperatorAddress())
	require.Equal(t, belowStakeAmountProofMissingPenalty.String(), expectedSlashingEvent.GetProofMissingPenalty())
}

func (s *TestSuite) TestClaimSettlement_ClaimSettled_ProofRequiredAndProvided_ViaProbability() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	// Use a single claim and proof for this test
	claim := s.claims[0]
	proof := s.proofs[0]
	relayMiningDifficulty := s.relayMiningDifficulties[0]

	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)

	// +1 so it's not required via probability
	proofRequirementThreshold = proofRequirementThreshold.Add(uPOKTCoin(1))

	// Set the proof parameters such that s.claim requires a proof because:
	// - proof_request_probability is 100%
	// - proof_requirement_threshold is 0, should not matter
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 1
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Upsert the claim & proof
	s.keepers.UpsertClaim(ctx, claim)
	s.keepers.UpsertProof(ctx, proof)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	_, _, err = s.keepers.ValidateSubmittedProofs(sdkCtx)
	require.NoError(t, err)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proof window has definitely closed at this point
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, expiredResults, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(1), settledResults.GetNumClaims()) // 1 claim settled
	require.Equal(t, uint64(0), expiredResults.GetNumClaims()) // 0 claims expired
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)      // 0 claims discarded

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Equal(t, 0, len(claims))

	// Confirm an settlement event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events)
	require.Equal(t, 1, len(expectedEvents))

	// Validate the settlement event
	expectedEvent := expectedEvents[0]
	require.Equal(t, prooftypes.ProofRequirementReason_PROBABILISTIC, expectedEvent.GetProofRequirement())
	require.Equal(t, s.numRelays, expectedEvent.GetNumRelays())
	require.Equal(t, s.numClaimedComputeUnits, expectedEvent.GetNumClaimedComputeUnits())
	require.Equal(t, s.numEstimatedComputeUnits, expectedEvent.GetNumEstimatedComputeUnits())
	require.Equal(t, s.claimedUpokt.String(), expectedEvent.GetClaimedUpokt())
}

func (s *TestSuite) TestSettlePendingClaims_Settles_WhenAProofIsNotRequired() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	// Use a single claim for this test
	claim := s.claims[0]
	relayMiningDifficulty := s.relayMiningDifficulties[0]

	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)

	// +1 to push threshold above s.claim's compute units
	proofRequirementThreshold = proofRequirementThreshold.Add(uPOKTCoin(1))

	// Set the proof parameters such that s.claim DOES NOT require a proof because:
	// - proof_request_probability is 0% AND
	// - proof_requirement_threshold exceeds s.claim's compute units
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Upsert the claim only (not the proof)
	s.keepers.UpsertClaim(ctx, claim)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	s.keepers.ValidateSubmittedProofs(sdkCtx)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, expiredResults, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that one claim was settled.
	require.Equal(t, uint64(1), settledResults.GetNumClaims()) // 1 claim settled
	require.Equal(t, uint64(0), expiredResults.GetNumClaims()) // 0 claims expired
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)      // 0 claims discarded

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Equal(t, 0, len(claims))

	// Confirm a settlement event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events)
	require.Equal(t, 1, len(expectedEvents))

	// Validate the settlement event
	expectedEvent := expectedEvents[0]
	require.Equal(t, prooftypes.ProofRequirementReason_NOT_REQUIRED.String(), expectedEvent.GetProofRequirement().String())
	require.Equal(t, s.numRelays, expectedEvent.GetNumRelays())
	require.Equal(t, s.numClaimedComputeUnits, expectedEvent.GetNumClaimedComputeUnits())
	require.Equal(t, s.numEstimatedComputeUnits, expectedEvent.GetNumEstimatedComputeUnits())
	require.Equal(t, s.claimedUpokt.String(), expectedEvent.GetClaimedUpokt())
}

func (s *TestSuite) TestSettlePendingClaims_ClaimDiscarded_WhenHasZeroSum() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	// Use a single claim for this test
	claim := s.claims[0]

	relayMiningDifficulty := s.relayMiningDifficulties[0]

	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)

	// Set the sum bytes of the claim's root hash to 0 to indicate a zero-sum claim.
	binary.BigEndian.PutUint64(claim.RootHash[protocol.TrieHasherSize:], 0)

	// Set the proof parameters such that s.claim DOES NOT require a proof because:
	// - proof_request_probability is 0% AND
	// - proof_requirement_threshold exceeds s.claim's compute units
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Upsert the claim only (not the proof)
	s.keepers.UpsertClaim(ctx, claim)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	s.keepers.ValidateSubmittedProofs(sdkCtx)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be ignored.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResults, expiredResults, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled or expired.
	require.Equal(t, uint64(0), settledResults.GetNumClaims()) // 0 claims settled
	require.Equal(t, uint64(0), expiredResults.GetNumClaims()) // 0 claims expired
	require.Equal(t, uint64(1), numDiscardedFaultyClaims)      // 1 claims discarded

	// Validate that the zero sum claim was deleted and no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Equal(t, 0, len(claims))
}

func (s *TestSuite) TestSettlePendingClaims_DoesNotSettle_BeforeProofWindowCloses() {
	s.T().Skip("TODO_TEST: Implement that a claim remains unsettled before the proof window closes")
}

func (s *TestSuite) TestSettlePendingClaims_DoesNotSettle_IfProofIsInvalid() {
	s.T().Skip("TODO_TEST: Implement that a claim remains unsettled before the proof window closes")
}

func (s *TestSuite) TestSettlePendingClaims_DoesNotSettle_IfProofIsRequiredButMissing() {
	s.T().Skip("TODO_TEST: Implement that a claim remains unsettled before the proof window closes")
}

func (s *TestSuite) TestSettlePendingClaims_MultipleClaimsSettle_WithMultipleApplicationsAndSuppliers() {
	s.T().Skip("TODO_TEST: Implement that multiple claims settle at once when different sessions have overlapping applications and suppliers")
}

func (s *TestSuite) TestSettlePendingClaims_ClaimPendingAfterSettlement() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	// Use a single claim for this test
	claim := s.claims[0]
	relayMiningDifficulty := s.relayMiningDifficulties[0]

	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)

	// +1 to push threshold above s.claim's compute units
	proofRequirementThreshold = proofRequirementThreshold.Add(uPOKTCoin(1))

	// Set the proof parameters such that s.claim DOES NOT require a proof
	// because the proof_request_probability is 0% and the proof_request_threshold
	// is greater than the claims' compute units.
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// 0. Add the claims & verify they exists
	sessionOneClaim := claim
	s.keepers.UpsertClaim(ctx, sessionOneClaim)

	sessionOneEndHeight := sessionOneClaim.GetSessionHeader().GetSessionEndBlockHeight()

	// Add a second claim with a session header corresponding to the next session.
	sessionTwoClaim := testproof.BaseClaim(
		sessionOneClaim.GetSessionHeader().GetServiceId(),
		sessionOneClaim.GetSessionHeader().GetApplicationAddress(),
		sessionOneClaim.GetSupplierOperatorAddress(),
		s.numRelays,
	)

	sessionOneProofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionOneEndHeight)
	sessionTwoStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, sessionOneProofWindowCloseHeight+1)
	sessionTwoProofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionTwoStartHeight)

	sessionTwoClaim.SessionHeader = &sessiontypes.SessionHeader{
		ApplicationAddress:      sessionOneClaim.GetSessionHeader().GetApplicationAddress(),
		ServiceId:               claim.GetSessionHeader().GetServiceId(),
		SessionId:               "session_two_id",
		SessionStartBlockHeight: sessionTwoStartHeight,
		SessionEndBlockHeight:   sharedtypes.GetSessionEndHeight(&sharedParams, sessionTwoStartHeight),
	}
	s.keepers.UpsertClaim(ctx, sessionTwoClaim)

	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Equalf(2, len(claims), "expected %d claims, got %d", 2, len(claims))

	// 1. Settle pending claims while the session is still active.
	// Expectations: No claims should be settled because the session is still ongoing
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionOneEndHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResults, expiredResults, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that one claim was settled.
	require.Equal(t, uint64(1), settledResults.GetNumClaims())

	// Validate that no claims expired.
	require.Equal(t, uint64(0), expiredResults.GetNumClaims())

	// Validate that no claims were discarded.
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)

	// Validate that one claim still remains.
	claims = s.keepers.GetAllClaims(ctx)
	require.Equal(t, 1, len(claims))

	// Calculate a block height which is within session two's proof window.
	blockHeight = (sessionTwoProofWindowCloseHeight - sessionTwoStartHeight) / 2

	// 2. Settle pending claims just after the session ended.
	// Expectations: Claims should not be settled because the proof window hasn't closed yet.
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResults, expiredResults, numDiscardedFaultyClaims, err = s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled or expired.
	require.Equal(t, uint64(0), settledResults.GetNumClaims())
	require.Equal(t, uint64(0), expiredResults.GetNumClaims())
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)

	// Validate that the claim still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Equal(t, 1, len(claims))
}

func (s *TestSuite) TestSettlePendingClaims_ClaimExpired_SupplierUnstaked() {
	// Number of expired claims to create
	numExpiredClaims := 3
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	// Use a single claim for this test
	claim := s.claims[0]
	serviceId := claim.GetSessionHeader().GetServiceId()
	relayMiningDifficulty := s.relayMiningDifficulties[0]

	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)

	// -1 to push threshold below s.claim's compute units
	proofRequirementThreshold = proofRequirementThreshold.Sub(uPOKTCoin(1))

	// Set the proof parameters such that s.claim requires a proof because:
	// - proof_request_probability is 0%
	// - proof_requirement_threshold is below the claim (i.e. claim is above threshold)
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	// Set the proof missing penalty to be equal to the supplier's stake to make
	// its stake below the minimum stake requirement and trigger an unstake.
	proofParams.ProofMissingPenalty = &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(supplierStakeAmt)}
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Creating multiple claims without proofs to test the claim expiration process:
	// - We're creating exactly numExpiredClaims applications to create numExpiredClaims claims
	// - All claims are for the same settlement period
	// - This allows us to verify that only a single unbonding event is emitted
	//   despite having multiple expired claims for the same supplier
	expiredClaimsMap := make(map[string]*prooftypes.Claim, numExpiredClaims)
	for range numExpiredClaims {
		appStake := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
		appAddr := sample.AccAddress()
		app := apptypes.Application{
			Address:        appAddr,
			Stake:          &appStake,
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: serviceId}},
		}
		s.keepers.SetApplication(s.ctx, app)

		// Get the session for the application/supplier pair which is expected
		// to be claimed and for which a valid proof would be accepted.
		sessionReq := &sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			ServiceId:          serviceId,
			BlockHeight:        1,
		}
		sessionRes, sessionErr := s.keepers.GetSession(sdkCtx, sessionReq)
		require.NoError(t, sessionErr)

		sessionHeader := sessionRes.Session.Header
		merkleRoot := testproof.SmstRootWithSumAndCount(1000, 1000)
		claim := testtree.NewClaim(t, claim.SupplierOperatorAddress, sessionHeader, merkleRoot)
		expiredClaimsMap[sessionHeader.SessionId] = claim
		s.keepers.UpsertClaim(ctx, *claim)
	}

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should expire.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	sessionProofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = sdkCtx.WithBlockHeight(sessionProofWindowCloseHeight)
	_, _, _, err = s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	upcomingSessionEndHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, sessionProofWindowCloseHeight) - 1

	// Slashing should have occurred and the supplier is unstaked but still unbonding.
	slashedSupplier, supplierFound := s.keepers.GetSupplier(sdkCtx, claim.SupplierOperatorAddress)
	require.True(t, supplierFound)
	require.Equal(t, math.NewInt(0), slashedSupplier.Stake.Amount)
	require.Equal(t, uint64(upcomingSessionEndHeight), slashedSupplier.UnstakeSessionEndHeight)
	require.True(t, slashedSupplier.IsUnbonding())

	events := sdkCtx.EventManager().Events()

	// Confirm that a slashing event was emitted
	slashingEvents := testutilevents.FilterEvents[*tokenomicstypes.EventSupplierSlashed](t, events)
	// A slashing event should be emitted for each expired claim.
	require.Equal(t, numExpiredClaims, len(slashingEvents))

	// Validate the slashing events
	for i, slashingEvent := range slashingEvents {
		proofMissingPenalty := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0)
		// The current slashing mechanism burns the whole supplier's stake, which
		// means that the first ocurrence will take the whole stake.
		if i == 0 {
			proofMissingPenalty = *s.keepers.ProofKeeper.GetParams(sdkCtx).ProofMissingPenalty
		}

		sessionId := slashingEvent.GetClaim().GetSessionHeader().GetSessionId()
		expectedSlashingEvent := &tokenomicstypes.EventSupplierSlashed{
			Claim:               expiredClaimsMap[sessionId],
			ProofMissingPenalty: proofMissingPenalty.String(),
		}
		require.EqualValues(t, expectedSlashingEvent, slashingEvents[i])
	}

	// Assert that an EventSupplierUnbondingBegin event was emitted.
	unbondingBeginEvents := testutilevents.FilterEvents[*suppliertypes.EventSupplierUnbondingBegin](t, events)
	require.Equal(t, 1, len(unbondingBeginEvents))

	// Validate the EventSupplierUnbondingBegin event.
	for i := len(slashedSupplier.ServiceConfigHistory) - 1; i >= 0; i-- {
		if slashedSupplier.ServiceConfigHistory[i].Service.ServiceId != serviceId {
			slashedSupplier.ServiceConfigHistory = append(
				slashedSupplier.ServiceConfigHistory[:i],
				slashedSupplier.ServiceConfigHistory[i+1:]...,
			)
			continue
		}
		slashedSupplier.ServiceConfigHistory[i].DeactivationHeight = upcomingSessionEndHeight
		if slashedSupplier.ServiceConfigHistory[i].Service.Endpoints == nil {
			slashedSupplier.ServiceConfigHistory[i].Service.Endpoints = []*sharedtypes.SupplierEndpoint{}
		}
	}
	// Get the active service configs at the time of the claimed session end height.
	slashedSupplier.Services = slashedSupplier.GetActiveServiceConfigs(sessionEndHeight)

	// DEV_NOTE: The slashing flow skips populating all the supplier's history for performance reasons.
	// The slashed supplier Services property already has the relevant active service configs at the time of the claimed session end height.
	slashedSupplier.ServiceConfigHistory = []*sharedtypes.ServiceConfigUpdate{}
	expectedUnbondingBeginEvent := &suppliertypes.EventSupplierUnbondingBegin{
		Supplier:           &slashedSupplier,
		Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE,
		SessionEndHeight:   upcomingSessionEndHeight,
		UnbondingEndHeight: upcomingSessionEndHeight,
	}
	// A single unbonding begin event corresponding to the slashed supplier should be
	// emitted for all expired claims.
	require.Len(t, unbondingBeginEvents, 1)
	require.EqualValues(t, expectedUnbondingBeginEvent, unbondingBeginEvents[0])

	// Advance the block height to the settlement session end height.
	settlementHeight := sharedtypes.GetSettlementSessionEndHeight(&sharedParams, sdkCtx.BlockHeight())
	sdkCtx.WithBlockHeight(settlementHeight)

	// Assert that the EventSupplierUnbondingEnd event is emitted.
	unbondingEndEvents := testutilevents.FilterEvents[*suppliertypes.EventSupplierUnbondingBegin](t, events)
	require.Equal(t, 1, len(unbondingEndEvents))

	// Validate the EventSupplierUnbondingEnd event.
	expectedUnbondingEndEvent := &suppliertypes.EventSupplierUnbondingEnd{
		Supplier:           &slashedSupplier,
		Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE,
		SessionEndHeight:   upcomingSessionEndHeight,
		UnbondingEndHeight: upcomingSessionEndHeight,
	}
	// A single unbonding end event corresponding to the slashed supplier should be
	// emitted for all expired claims.
	require.Len(t, unbondingEndEvents, 1)
	require.EqualValues(t, expectedUnbondingEndEvent, unbondingEndEvents[0])
}

func (s *TestSuite) TestSettlePendingClaims_MultipleClaimsFromDifferentServices() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// All claims have the same session end height, use the first one
	sessionEndHeight := s.claims[0].SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)

	// All claims have the same proof requirement threshold, pick the first one
	proofRequirementThreshold, err := s.claims[0].GetClaimeduPOKT(sharedParams, s.relayMiningDifficulties[0])
	require.NoError(t, err)

	// -1 to push threshold below s.claim's compute units
	proofRequirementThreshold = proofRequirementThreshold.Sub(uPOKTCoin(1))

	// Set the proof parameters such that s.claim requires a proof because:
	// - proof_request_probability is 0%
	// - proof_requirement_threshold is below the claim (i.e. claim is above threshold)
	err = s.keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability:   0,
		ProofRequirementThreshold: &proofRequirementThreshold,
	})
	require.NoError(t, err)

	// Upsert the claims
	for _, claim := range s.claims {
		s.keepers.UpsertClaim(ctx, claim)
	}

	// Upsert the proofs
	for _, proof := range s.proofs {
		s.keepers.UpsertProof(ctx, proof)
	}

	s.keepers.ValidateSubmittedProofs(sdkCtx)

	// Settle pending claims after proof window closes
	// Expectation: All claims should be claimed.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	settledResult, expiredResult, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(len(s.claims)), settledResult.GetNumClaims())
	require.Equal(t, uint64(0), expiredResult.GetNumClaims())
	require.Equal(t, uint64(0), numDiscardedFaultyClaims) // No faulty claims discarded

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Equal(t, 0, len(claims))

	// Confirm settlement events were emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events)
	require.Equal(t, len(s.claims), len(expectedEvents))

	// Validate the events
	for _, expectedEvent := range expectedEvents {
		require.Equal(t, prooftypes.ProofRequirementReason_THRESHOLD, expectedEvent.GetProofRequirement())
		require.Equal(t, s.numRelays, expectedEvent.GetNumRelays())
		require.Equal(t, s.numClaimedComputeUnits, expectedEvent.GetNumClaimedComputeUnits())
		require.Equal(t, s.numEstimatedComputeUnits, expectedEvent.GetNumEstimatedComputeUnits())
		require.Equal(t, s.claimedUpokt.String(), expectedEvent.GetClaimedUpokt())
	}
}

// getEstimatedComputeUnits returns the estimated number of compute units given
// the number of claimed compute units and the relay mining difficulty.
func getEstimatedComputeUnits(
	numClaimedComputeUnits uint64,
	relayMiningDifficulty servicetypes.RelayMiningDifficulty,
) uint64 {
	difficultyMultiplierRat := protocol.GetRelayDifficultyMultiplier(relayMiningDifficulty.GetTargetHash())
	numClaimedComputeUnitsRat := new(big.Rat).SetUint64(numClaimedComputeUnits)
	numEstimatedComputeUnitsRat := new(big.Rat).Mul(difficultyMultiplierRat, numClaimedComputeUnitsRat)

	return new(big.Int).Div(numEstimatedComputeUnitsRat.Num(), numEstimatedComputeUnitsRat.Denom()).Uint64()
}

// getClaimedUpokt returns the claimed amount in uPOKT.
func getClaimedUpokt(
	sharedParams sharedtypes.Params,
	numClaimedComputeUnits uint64,
	relayMiningDifficulty servicetypes.RelayMiningDifficulty,
) cosmostypes.Coin {
	// Calculate the number of estimated compute units ratio instead of directly using
	// the integer value to avoid precision loss.
	difficultyMultiplierRat := protocol.GetRelayDifficultyMultiplier(relayMiningDifficulty.GetTargetHash())
	numClaimedComputeUnitsRat := new(big.Rat).SetUint64(numClaimedComputeUnits)
	numEstimatedComputeUnitsRat := new(big.Rat).Mul(difficultyMultiplierRat, numClaimedComputeUnitsRat)

	computeUnitsToTokensMultiplierRat := new(big.Rat).SetFrac64(
		int64(sharedParams.GetComputeUnitsToTokensMultiplier()),
		int64(sharedParams.GetComputeUnitCostGranularity()),
	)

	claimedUpoktRat := new(big.Rat).Mul(numEstimatedComputeUnitsRat, computeUnitsToTokensMultiplierRat)
	claimedUpoktInt := new(big.Int).Div(claimedUpoktRat.Num(), claimedUpoktRat.Denom())

	return cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewIntFromBigInt(claimedUpoktInt))
}

// uPOKTCoin returns a uPOKT coin with the given amount.
func uPOKTCoin(amount int64) cosmostypes.Coin {
	return cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewInt(amount))
}

// createTestActors sets up the necessary test actors (applications and a supplier) with
// the specified services and stakes in the respective keepers.
//
// - It creates a supplier with a stake amount significantly more than the minimum
// - Creates applications for each service ID defined in testServiceIds
// - Each application is staked to its respective service.
// - The function returns the application addresses and the supplier's owner address.
func (s *TestSuite) createTestActors(
	t *testing.T,
	ctx cosmostypes.Context,
	keyRing keyring.Keyring,
) (appAddresses []string, supplierOwnerAddr string) {
	// Create a pre-generated account iterator to create accounts for the test.
	preGeneratedAccts := testkeyring.PreGeneratedAccounts()

	// Create accounts in the account keeper with corresponding keys in the keyring
	// // for the applications and suppliers used in the tests.
	supplierOwnerAddr = testkeyring.CreateOnChainAccount(
		ctx, t,
		"supplier",
		keyRing,
		s.keepers.AccountKeeper,
		preGeneratedAccts,
	).String()

	appStake := cosmostypes.NewCoin("upokt", math.NewInt(1000000))

	// Setup the test for each service:
	// - Create and store the service in the service keeper.
	// - Create and store an application staked to the service.
	// - Create a supplier service config for the service.
	supplierServiceConfigs := make([]*sharedtypes.SupplierServiceConfig, 0, len(testServiceIds))
	appAddresses = make([]string, 0, len(testServiceIds))
	for i, serviceId := range testServiceIds {
		service := sharedtypes.Service{
			Id:                   serviceId,
			ComputeUnitsPerRelay: computeUnitsPerRelay,
			OwnerAddress:         sample.AccAddress(),
		}
		s.keepers.SetService(s.ctx, service)

		supplierServiceConfigs = append(
			supplierServiceConfigs,
			&sharedtypes.SupplierServiceConfig{
				ServiceId: serviceId,
				RevShare: []*sharedtypes.ServiceRevenueShare{{
					Address:            supplierOwnerAddr,
					RevSharePercentage: 100,
				}},
			},
		)

		appAddr := testkeyring.CreateOnChainAccount(
			ctx, t,
			fmt.Sprintf("app%d", i),
			keyRing,
			s.keepers.AccountKeeper,
			preGeneratedAccts,
		).String()
		appAddresses = append(appAddresses, appAddr)

		app := apptypes.Application{
			Address:        appAddr,
			Stake:          &appStake,
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: serviceId}},
		}
		s.keepers.SetApplication(s.ctx, app)
	}

	// Make the supplier staked for each tested service.
	supplierStake := cosmostypes.NewCoin("upokt", math.NewInt(supplierStakeAmt))
	supplierServiceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(supplierOwnerAddr, supplierServiceConfigs, 1, 0)
	supplier := sharedtypes.Supplier{
		OwnerAddress:         supplierOwnerAddr,
		OperatorAddress:      supplierOwnerAddr,
		Stake:                &supplierStake,
		Services:             supplierServiceConfigs,
		ServiceConfigHistory: supplierServiceConfigHistory,
	}
	s.keepers.SetAndIndexDehydratedSupplier(s.ctx, supplier)

	return appAddresses, supplierOwnerAddr
}

// createTestClaimsAndProofs generates test claims and corresponding proofs for session.
//
// For each service in testServiceIds, it:
// 1. Retrieves the session for the application/supplier pair
// 2. Creates a session tree with the specified number of relays
// 3. Calculates the claimed and estimated compute units
// 4. Prepares the claims and proofs with proper merkle roots and paths
func (s *TestSuite) createTestClaimsAndProofs(
	t *testing.T,
	ctx cosmostypes.Context,
	appAddresses []string,
	supplierOwnerAddr string,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) (claims []prooftypes.Claim, proofs []prooftypes.Proof) {
	// Create a claim and proof for each service.
	s.relayMiningDifficulties = make([]servicetypes.RelayMiningDifficulty, 0, len(testServiceIds))
	for i, serviceId := range testServiceIds {
		appAddress := appAddresses[i]
		// Get the session for the application/supplier pair which is expected
		// to be claimed and for which a valid proof would be accepted.
		sessionReq := &sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddress,
			ServiceId:          serviceId,
			BlockHeight:        1,
		}
		sessionRes, err := s.keepers.GetSession(ctx, sessionReq)
		require.NoError(t, err)
		sessionHeader := sessionRes.Session.Header

		// Construct a valid session tree with 100 relays.
		s.numRelays = uint64(100)
		sessionTree := testtree.NewFilledSessionTree(
			ctx, t,
			s.numRelays, computeUnitsPerRelay,
			"supplier", supplierOwnerAddr,
			sessionHeader, sessionHeader, sessionHeader,
			keyRing,
			ringClient,
		)

		// Calculate the number of claimed compute units.
		s.numClaimedComputeUnits = s.numRelays * computeUnitsPerRelay

		targetNumRelays := s.keepers.ServiceKeeper.GetParams(ctx).TargetNumRelays
		relayMiningDifficulty := servicekeeper.NewDefaultRelayMiningDifficulty(
			ctx,
			s.keepers.Logger(),
			serviceId,
			targetNumRelays,
			targetNumRelays,
		)

		s.relayMiningDifficulties = append(
			s.relayMiningDifficulties,
			relayMiningDifficulty,
		)

		// Calculate the number of estimated compute units.
		s.numEstimatedComputeUnits = getEstimatedComputeUnits(s.numClaimedComputeUnits, relayMiningDifficulty)

		// Calculate the claimed amount in uPOKT.
		sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
		s.claimedUpokt = getClaimedUpokt(sharedParams, s.numEstimatedComputeUnits, relayMiningDifficulty)

		blockHeaderHash := make([]byte, 0)
		expectedMerkleProofPath := protocol.GetPathForProof(blockHeaderHash, sessionHeader.SessionId)

		// Advance the block height to the earliest claim commit height.
		claimMsgHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
			&sharedParams,
			sessionHeader.GetSessionEndBlockHeight(),
			blockHeaderHash,
			supplierOwnerAddr,
		)
		ctx = ctx.WithBlockHeight(claimMsgHeight).WithHeaderHash(blockHeaderHash)
		s.ctx = ctx

		merkleRootBz, err := sessionTree.Flush()
		require.NoError(t, err)

		// Prepare a claim that can be inserted
		claims = append(
			claims,
			*testtree.NewClaim(t, supplierOwnerAddr, sessionHeader, merkleRootBz),
		)

		proofs = append(
			proofs,
			*testtree.NewProof(t, supplierOwnerAddr, sessionHeader, sessionTree, expectedMerkleProofPath),
		)
	}

	return claims, proofs
}

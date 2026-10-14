package keeper_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"
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
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	supplierkeeper "github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const computeUnitsPerRelay = 1

var (
	// Test settlements with claims from multiple services
	testServiceIds   = []string{"svc1", "svc2"}
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

	// Set up a known proposer and validator for testing
	proposerConsAddr := sample.ConsAddress()
	proposerValOperatorAddr := sample.ValOperatorAddress()

	moduleBalancesOpt := keepertest.WithModuleAccountBalances(map[string]int64{
		apptypes.ModuleName:      1000000000,
		suppliertypes.ModuleName: supplierStakeAmt,
	})
	proposerOpt := keepertest.WithBlockProposer(proposerConsAddr, proposerValOperatorAddr)
	s.keepers, s.ctx = keepertest.NewTokenomicsModuleKeepers(s.T(), nil, moduleBalancesOpt, proposerOpt)
	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(1)
	// Update s.ctx with the modified context
	s.ctx = sdkCtx

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(s.keepers.Codec)

	// Construct a ringClient to get the application's ring & verify the relay
	// request signature.
	ringClient, err := rings.NewRingClient(depinject.Supply(
		polyzero.NewLogger(polyzero.WithLevel(polyzero.ErrorLevel)),
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

	// The event no longer contains the full claim, so we validate the individual fields
	require.Equal(t, s.claims[0].SessionHeader.ServiceId, expectedSlashingEvent.GetServiceId())
	require.Equal(t, s.claims[0].SessionHeader.ApplicationAddress, expectedSlashingEvent.GetApplicationAddress())
	require.Equal(t, s.claims[0].SessionHeader.SessionEndBlockHeight, expectedSlashingEvent.GetSessionEndBlockHeight())
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
	require.Equal(t, int32(prooftypes.ProofRequirementReason_THRESHOLD), expectedEvent.GetProofRequirementInt())
	require.Equal(t, s.numRelays, expectedEvent.GetNumRelays())
	require.Equal(t, s.numClaimedComputeUnits, expectedEvent.GetNumClaimedComputeUnits())
	require.Equal(t, s.numEstimatedComputeUnits, expectedEvent.GetNumEstimatedComputeUnits())
	require.Equal(t, s.claimedUpokt.String(), expectedEvent.GetClaimedUpokt())

	// Validate reward distribution is not empty
	// DEV_NOTE: DOES NOT validate reward distribution amounts, this is out of scope for this test.
	rewardDistribution := expectedEvent.GetRewardDistribution()
	require.NotNil(t, rewardDistribution, "reward distribution should not be nil")
	require.NotEmpty(t, rewardDistribution, "reward distribution should not be empty")
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
	require.Equal(t, claim.SessionHeader.ServiceId, expectedProofValidityCheckedEvent.GetServiceId())
	require.Equal(t, claim.SessionHeader.ApplicationAddress, expectedProofValidityCheckedEvent.GetApplicationAddress())
	require.Equal(t, claim.SessionHeader.SessionEndBlockHeight, expectedProofValidityCheckedEvent.GetSessionEndBlockHeight())
	// After ValidateSubmittedProofs, the claim should have INVALID status (2)
	require.Equal(t, int32(prooftypes.ClaimProofStatus_INVALID), expectedProofValidityCheckedEvent.GetClaimProofStatusInt())

	// Confirm that a slashing event was emitted
	expectedSlashingEvents := testutilevents.FilterEvents[*tokenomicstypes.EventSupplierSlashed](t, events)
	require.Equal(t, 1, len(expectedSlashingEvents))

	// Validate the slashing event
	expectedSlashingEvent := expectedSlashingEvents[0]
	// The event no longer contains the full claim, so we validate the individual fields
	require.Equal(t, s.claims[0].SessionHeader.ServiceId, expectedSlashingEvent.GetServiceId())
	require.Equal(t, s.claims[0].SessionHeader.ApplicationAddress, expectedSlashingEvent.GetApplicationAddress())
	require.Equal(t, s.claims[0].SessionHeader.SessionEndBlockHeight, expectedSlashingEvent.GetSessionEndBlockHeight())
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
	require.Equal(t, int32(prooftypes.ProofRequirementReason_PROBABILISTIC), expectedEvent.GetProofRequirementInt())
	require.Equal(t, s.numRelays, expectedEvent.GetNumRelays())
	require.Equal(t, s.numClaimedComputeUnits, expectedEvent.GetNumClaimedComputeUnits())
	require.Equal(t, s.numEstimatedComputeUnits, expectedEvent.GetNumEstimatedComputeUnits())
	require.Equal(t, s.claimedUpokt.String(), expectedEvent.GetClaimedUpokt())

	// Validate reward distribution is not empty
	// DEV_NOTE: DOES NOT validate reward distribution amounts, this is out of scope for this test.
	rewardDistribution := expectedEvent.GetRewardDistribution()
	require.NotNil(t, rewardDistribution, "reward distribution should not be nil")
	require.NotEmpty(t, rewardDistribution, "reward distribution should not be empty")
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
	require.Equal(t, int32(prooftypes.ProofRequirementReason_NOT_REQUIRED), expectedEvent.GetProofRequirementInt())
	require.Equal(t, s.numRelays, expectedEvent.GetNumRelays())
	require.Equal(t, s.numClaimedComputeUnits, expectedEvent.GetNumClaimedComputeUnits())
	require.Equal(t, s.numEstimatedComputeUnits, expectedEvent.GetNumEstimatedComputeUnits())
	require.Equal(t, s.claimedUpokt.String(), expectedEvent.GetClaimedUpokt())

	// Validate reward distribution is not empty
	// DEV_NOTE: DOES NOT validate reward distribution amounts, this is out of scope for this test.
	rewardDistribution := expectedEvent.GetRewardDistribution()
	require.NotNil(t, rewardDistribution, "reward distribution should not be nil")
	require.NotEmpty(t, rewardDistribution, "reward distribution should not be empty")
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
	// Tests multi-claim aggregation with real bank operations.
	// Setup: 3 applications (1 from SetupTest + 2 additional), 1 supplier, 1 service.
	// Each application has a claim for the same session.
	// Validates: aggregated settlement, EventClaimSettled per claim, EventSettlementBatch aggregation.
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Use the first claim from setup as a template.
	claim := s.claims[0]
	serviceId := claim.SessionHeader.ServiceId
	supplierOpAddr := claim.SupplierOperatorAddress

	// Set proof params so no proof is required.
	relayMiningDifficulty := s.relayMiningDifficulties[0]
	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)
	proofRequirementThreshold = proofRequirementThreshold.Add(uPOKTCoin(1))

	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Upsert the first claim from setup.
	s.keepers.UpsertClaim(ctx, claim)

	// Create 2 additional applications and claims for the same service.
	for i := 0; i < 2; i++ {
		appStake := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
		appAddr := sample.AccAddressBech32()
		app := apptypes.Application{
			Address:        appAddr,
			Stake:          &appStake,
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: serviceId}},
		}
		s.keepers.SetApplication(ctx, app)

		sessionReq := &sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			ServiceId:          serviceId,
			BlockHeight:        1,
		}
		sessionRes, sessionErr := s.keepers.GetSession(sdkCtx, sessionReq)
		require.NoError(t, sessionErr)

		merkleRoot := testproof.SmstRootWithSumAndCount(5, 5)
		extraClaim := testtree.NewClaim(t, supplierOpAddr, sessionRes.Session.Header, merkleRoot)
		s.keepers.UpsertClaim(ctx, *extraClaim)
	}

	// Advance to proof window close and settle.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, expiredResults, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// All 3 claims should settle.
	require.Equal(t, uint64(3), settledResults.GetNumClaims())
	require.Equal(t, uint64(0), expiredResults.GetNumClaims())
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)

	// 3 EventClaimSettled events should be emitted.
	events := sdkCtx.EventManager().Events()
	claimSettledEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events)
	require.Equal(t, 3, len(claimSettledEvents))

	// EventSettlementBatch events should be present.
	batchEvents := testutilevents.FilterEvents[*tokenomicstypes.EventSettlementBatch](t, events)
	require.NotEmpty(t, batchEvents)

	// The number of batch events should be much fewer than 3 * ops_per_claim,
	// since claims sharing the same aggregation keys are batched together.
	// Each batch event should have NumClaims >= 1.
	for _, batchEvt := range batchEvents {
		require.True(t, batchEvt.NumClaims >= 1, "batch event should have at least 1 claim")
	}

	// Verify batch events have NumClaims == 3 for keys shared by all 3 claims.
	// At minimum, the supplier stake mint and application stake burn keys are shared.
	hasThreeClaimBatch := false
	for _, batchEvt := range batchEvents {
		if batchEvt.NumClaims == 3 {
			hasThreeClaimBatch = true
			break
		}
	}
	require.True(t, hasThreeClaimBatch, "at least one batch event should aggregate all 3 claims")
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
		appAddr := sample.AccAddressBech32()
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
	unbondingEndHeight := sharedtypes.GetSupplierUnbondingEndHeight(&sharedParams, &slashedSupplier)

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

		// The event no longer contains the full claim, so we construct the expected event with individual fields.
		// The penalty equals the full stake on the first occurrence and 0 afterwards,
		// so the supplier's post-slash stake is 0upokt for every slashing event here.
		expectedSlashingEvent := &tokenomicstypes.EventSupplierSlashed{
			ProofMissingPenalty:     proofMissingPenalty.String(),
			SupplierStakeAfterSlash: cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 0).String(),
			ServiceId:               slashingEvent.GetServiceId(),
			ApplicationAddress:      slashingEvent.GetApplicationAddress(),
			SessionEndBlockHeight:   slashingEvent.GetSessionEndBlockHeight(),
			ClaimProofStatusInt:     slashingEvent.GetClaimProofStatusInt(),
			SupplierOperatorAddress: slashingEvent.GetSupplierOperatorAddress(),
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
		UnbondingEndHeight: unbondingEndHeight,
	}
	// A single unbonding begin event corresponding to the slashed supplier should be
	// emitted for all expired claims.
	require.Len(t, unbondingBeginEvents, 1)
	require.EqualValues(t, expectedUnbondingBeginEvent, unbondingBeginEvents[0])
}

// TestSettlePendingClaims_SlashBelowMinStake_UnbondsSupplier asserts that a
// supplier slashed below the min stake during settlement is unbonded end to end:
// it keeps its current session, leaves every session from the next one on, and
// is removed once its unbonding period elapses. The settlement context only holds
// a dehydrated supplier, so this guards against persisting it without its
// service config and unstaking indexes.
func (s *TestSuite) TestSettlePendingClaims_SlashBelowMinStake_UnbondsSupplier() {
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(1)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	claim := s.claims[0]
	serviceId := claim.GetSessionHeader().GetServiceId()
	supplierOperatorAddr := claim.SupplierOperatorAddress

	// Require a proof for the claim and make the missing proof penalty the whole stake.
	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulties[0])
	require.NoError(t, err)
	proofRequirementThreshold = proofRequirementThreshold.Sub(uPOKTCoin(1))
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	proofParams.ProofMissingPenalty = &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(supplierStakeAmt)}
	require.NoError(t, s.keepers.ProofKeeper.SetParams(ctx, proofParams))

	appStake := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
	appAddr := sample.AccAddressBech32()
	s.keepers.SetApplication(s.ctx, apptypes.Application{
		Address:        appAddr,
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: serviceId}},
	})
	getSessionSuppliers := func(ctx cosmostypes.Context, height int64) []string {
		sessionRes, sessionErr := s.keepers.GetSession(ctx.WithBlockHeight(height), &sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			ServiceId:          serviceId,
			BlockHeight:        height,
		})
		// The slashed supplier is the fixture's only supplier for the service.
		if sessionErr != nil && strings.Contains(sessionErr.Error(), sessiontypes.ErrSessionSuppliersNotFound.Error()) {
			return nil
		}
		require.NoError(t, sessionErr)
		operatorAddrs := make([]string, 0, len(sessionRes.GetSession().GetSuppliers()))
		for _, supplier := range sessionRes.GetSession().GetSuppliers() {
			operatorAddrs = append(operatorAddrs, supplier.GetOperatorAddress())
		}
		return operatorAddrs
	}

	sessionRes, err := s.keepers.GetSession(sdkCtx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddr,
		ServiceId:          serviceId,
		BlockHeight:        1,
	})
	require.NoError(t, err)
	merkleRoot := testproof.SmstRootWithSumAndCount(1000, 1000)
	s.keepers.UpsertClaim(ctx, *testtree.NewClaim(t, supplierOperatorAddr, sessionRes.Session.Header, merkleRoot))

	// The claim expires without a proof: the supplier is slashed to 0 and force unstaked.
	sessionProofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, claim.SessionHeader.SessionEndBlockHeight)
	sdkCtx = sdkCtx.WithBlockHeight(sessionProofWindowCloseHeight)
	_, _, _, err = s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	unstakeSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sessionProofWindowCloseHeight)
	nextSessionStartHeight := unstakeSessionEndHeight + 1

	slashedSupplier, found := s.keepers.GetSupplier(sdkCtx, supplierOperatorAddr)
	require.True(t, found)
	require.Equal(t, uint64(unstakeSessionEndHeight), slashedSupplier.UnstakeSessionEndHeight)
	require.NotEmpty(t, slashedSupplier.ServiceConfigHistory, "service configs MUST survive re-indexing")
	for _, serviceConfig := range slashedSupplier.ServiceConfigHistory {
		require.Equal(t, nextSessionStartHeight, serviceConfig.DeactivationHeight)
	}

	// The supplier keeps its current session and leaves the next one.
	currentSessionStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, sessionProofWindowCloseHeight)
	require.Contains(t, getSessionSuppliers(sdkCtx, currentSessionStartHeight), supplierOperatorAddr)
	require.NotContains(t, getSessionSuppliers(sdkCtx, nextSessionStartHeight), supplierOperatorAddr)

	// The begin event announces the height at which the supplier is actually unbonded.
	unbondingEndHeight := sharedtypes.GetSupplierUnbondingEndHeight(&sharedParams, &slashedSupplier)
	unbondingBeginEvents := testutilevents.FilterEvents[*suppliertypes.EventSupplierUnbondingBegin](t, sdkCtx.EventManager().Events())
	require.Len(t, unbondingBeginEvents, 1)
	require.Equal(t, unbondingEndHeight, unbondingBeginEvents[0].GetUnbondingEndHeight())

	// The supplier is unbonded once its unbonding period elapses.
	supplierKeeper := s.keepers.SupplierKeeper.(*supplierkeeper.Keeper)
	unbondingEndCtx := sdkCtx.WithBlockHeight(unbondingEndHeight).WithEventManager(cosmostypes.NewEventManager())
	numUnbondedSuppliers, err := supplierKeeper.EndBlockerUnbondSuppliers(unbondingEndCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numUnbondedSuppliers)
	_, found = supplierKeeper.GetDehydratedSupplier(sdkCtx, supplierOperatorAddr)
	require.False(t, found)

	unbondingEndEvents := testutilevents.FilterEvents[*suppliertypes.EventSupplierUnbondingEnd](t, unbondingEndCtx.EventManager().Events())
	require.Len(t, unbondingEndEvents, 1)
	require.Equal(t, supplierOperatorAddr, unbondingEndEvents[0].GetSupplier().GetOperatorAddress())
	require.Equal(t, suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE, unbondingEndEvents[0].GetReason())
	require.Equal(t, unbondingEndHeight, unbondingEndEvents[0].GetUnbondingEndHeight())
}

// setupSlashCapScenario prepares a settlement in which every expired claim of the
// fixture supplier is missing its proof and costs `price` of stake. The supplier
// is re-staked with services [claim service, extras...] in that priority order,
// with the service cap set so that stake covers all of them (1 included service,
// then one more per `price` above min_stake; min_stake = stake - 2*price, so 3 services are covered).
// It returns the block height at which the claims expire.
func (s *TestSuite) setupSlashCapScenario(price int64, extraServiceIds []string, capEnabled bool) (settleHeight int64, serviceId, supplierOperatorAddr string) {
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	claim := s.claims[0]
	serviceId = claim.GetSessionHeader().GetServiceId()
	supplierOperatorAddr = claim.SupplierOperatorAddress

	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulties[0])
	require.NoError(t, err)
	proofRequirementThreshold = proofRequirementThreshold.Sub(uPOKTCoin(1))
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	proofParams.ProofMissingPenalty = &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(price)}
	require.NoError(t, s.keepers.ProofKeeper.SetParams(ctx, proofParams))

	// Re-stake the supplier with the claim's service first, then the extras.
	supplier, found := s.keepers.GetSupplier(ctx, supplierOperatorAddr)
	require.True(t, found)
	serviceIds := append([]string{serviceId}, extraServiceIds...)
	configs := make([]*sharedtypes.SupplierServiceConfig, 0, len(serviceIds))
	for _, id := range serviceIds {
		s.keepers.SetService(ctx, sharedtypes.Service{Id: id, ComputeUnitsPerRelay: computeUnitsPerRelay, OwnerAddress: sample.AccAddressBech32()})
		configs = append(configs, &sharedtypes.SupplierServiceConfig{
			ServiceId: id,
			RevShare:  []*sharedtypes.ServiceRevenueShare{{Address: supplier.OwnerAddress, RevSharePercentage: 100}},
		})
	}
	supplier.Services = configs
	supplier.ServiceConfigHistory = sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(supplierOperatorAddr, configs, 1, 0)
	s.keepers.SetAndIndexDehydratedSupplier(ctx, supplier)
	s.keepers.SupplierKeeper.(*supplierkeeper.Keeper).SetServicePriority(ctx, supplierOperatorAddr, serviceIds)

	if capEnabled {
		supplierKeeper := s.keepers.SupplierKeeper.(*supplierkeeper.Keeper)
		supplierParams := supplierKeeper.GetParams(ctx)
		minStake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, supplierStakeAmt-2*price)
		priceCoin := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, price)
		supplierParams.MinStake = &minStake
		supplierParams.MinStakeIncludedServices = 1
		supplierParams.StakePerAdditionalService = &priceCoin
		require.NoError(t, supplierKeeper.SetParams(ctx, supplierParams))
	}

	return sharedtypes.GetProofWindowCloseHeight(&sharedParams, claim.SessionHeader.SessionEndBlockHeight), serviceId, supplierOperatorAddr
}

// upsertMissingProofClaim creates a new app for the service and upserts a claim
// (without a proof) for the supplier in that app's session.
func (s *TestSuite) upsertMissingProofClaim(serviceId, supplierOperatorAddr string) (appAddr string) {
	t := s.T()
	appStake := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
	appAddr = sample.AccAddressBech32()
	s.keepers.SetApplication(s.ctx, apptypes.Application{
		Address:        appAddr,
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: serviceId}},
	})
	sessionRes, err := s.keepers.GetSession(cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(1), &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddr,
		ServiceId:          serviceId,
		BlockHeight:        1,
	})
	require.NoError(t, err)
	merkleRoot := testproof.SmstRootWithSumAndCount(1000, 1000)
	s.keepers.UpsertClaim(s.ctx, *testtree.NewClaim(t, supplierOperatorAddr, sessionRes.Session.Header, merkleRoot))
	return appAddr
}

// deactivationHeights returns the deactivation height of every deactivated service config.
func deactivationHeights(supplier sharedtypes.Supplier) map[string]int64 {
	deactivatedAt := make(map[string]int64)
	for _, config := range supplier.ServiceConfigHistory {
		if config.DeactivationHeight != sharedtypes.NoDeactivationHeight {
			deactivatedAt[config.Service.ServiceId] = config.DeactivationHeight
		}
	}
	return deactivatedAt
}

func (s *TestSuite) TestSettlePendingClaims_Slash_CapsServices() {
	t := s.T()
	sharedParams := s.keepers.SharedKeeper.GetParams(s.ctx)
	price := supplierStakeAmt / 10
	settleHeight, serviceId, supplierOperatorAddr := s.setupSlashCapScenario(price, []string{"svc_extra_1", "svc_extra_2"}, true)
	appAddr := s.upsertMissingProofClaim(serviceId, supplierOperatorAddr)

	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(settleHeight)
	_, _, _, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// One slash: stake covers 2 of 3 services; the last listed service is deactivated
	// at the next session start and the supplier is NOT unbonding.
	slashed, found := s.keepers.GetSupplier(sdkCtx, supplierOperatorAddr)
	require.True(t, found)
	require.False(t, slashed.IsUnbonding())
	next := sharedtypes.GetNextSessionStartHeight(&sharedParams, settleHeight)
	require.Equal(t, map[string]int64{"svc_extra_2": next}, deactivationHeights(slashed))

	capped := testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, sdkCtx.EventManager().Events())
	require.Len(t, capped, 1)
	require.Equal(t, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_SLASH, capped[0].GetReason())

	// svc_extra_2 is still active now and inactive from the next session start.
	activeServiceIds := func(height int64) []string {
		ids := make([]string, 0)
		for _, config := range slashed.GetActiveServiceConfigs(height) {
			ids = append(ids, config.GetServiceId())
		}
		return ids
	}
	require.Contains(t, activeServiceIds(settleHeight), "svc_extra_2")
	require.NotContains(t, activeServiceIds(next), "svc_extra_2")

	// The claimed session is unaffected: hydrating it still includes the supplier.
	currentSessionStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, settleHeight)
	sessionRes, err := s.keepers.GetSession(sdkCtx.WithBlockHeight(currentSessionStartHeight), &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddr,
		ServiceId:          serviceId,
		BlockHeight:        currentSessionStartHeight,
	})
	require.NoError(t, err)
	operatorAddrs := make([]string, 0, len(sessionRes.GetSession().GetSuppliers()))
	for _, supplier := range sessionRes.GetSession().GetSuppliers() {
		operatorAddrs = append(operatorAddrs, supplier.GetOperatorAddress())
	}
	require.Contains(t, operatorAddrs, supplierOperatorAddr)
}

func (s *TestSuite) TestSettlePendingClaims_TwoSlashesSameBlock_CapsTwice() {
	t := s.T()
	sharedParams := s.keepers.SharedKeeper.GetParams(s.ctx)
	price := supplierStakeAmt / 10
	settleHeight, serviceId, supplierOperatorAddr := s.setupSlashCapScenario(price, []string{"svc_extra_1", "svc_extra_2"}, true)
	before, found := s.keepers.GetSupplier(s.ctx, supplierOperatorAddr)
	require.True(t, found)
	s.upsertMissingProofClaim(serviceId, supplierOperatorAddr)
	s.upsertMissingProofClaim(serviceId, supplierOperatorAddr)

	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(settleHeight)
	_, _, _, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	slashed, found := s.keepers.GetSupplier(sdkCtx, supplierOperatorAddr)
	require.True(t, found)
	require.False(t, slashed.IsUnbonding())
	require.Equal(t, before.Stake.Amount.Int64()-2*price, slashed.Stake.Amount.Int64())
	next := sharedtypes.GetNextSessionStartHeight(&sharedParams, settleHeight)
	require.Equal(t, map[string]int64{"svc_extra_1": next, "svc_extra_2": next}, deactivationHeights(slashed))

	capped := testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, sdkCtx.EventManager().Events())
	require.Len(t, capped, 2)
	require.Equal(t, "svc_extra_2", capped[0].GetDeactivatedServiceId())
	require.Equal(t, "svc_extra_1", capped[1].GetDeactivatedServiceId())
}

func (s *TestSuite) TestSettlePendingClaims_Slash_UnderCap_Unchanged() {
	t := s.T()
	price := supplierStakeAmt / 10
	settleHeight, serviceId, supplierOperatorAddr := s.setupSlashCapScenario(price, nil, true)
	s.upsertMissingProofClaim(serviceId, supplierOperatorAddr)

	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(settleHeight)
	_, _, _, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	slashed, found := s.keepers.GetSupplier(sdkCtx, supplierOperatorAddr)
	require.True(t, found)
	require.Empty(t, deactivationHeights(slashed))
	require.Empty(t, testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, sdkCtx.EventManager().Events()))
}

func (s *TestSuite) TestSettlePendingClaims_Slash_CapDisabled_Unchanged() {
	t := s.T()
	price := supplierStakeAmt / 10
	settleHeight, serviceId, supplierOperatorAddr := s.setupSlashCapScenario(price, []string{"svc_extra_1", "svc_extra_2"}, false)
	s.upsertMissingProofClaim(serviceId, supplierOperatorAddr)

	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(settleHeight)
	_, _, _, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	slashed, found := s.keepers.GetSupplier(sdkCtx, supplierOperatorAddr)
	require.True(t, found)
	require.False(t, slashed.IsUnbonding())
	require.Less(t, slashed.Stake.Amount.Int64(), supplierStakeAmt, "the supplier MUST have been slashed")
	require.Empty(t, deactivationHeights(slashed))
	require.Empty(t, testutilevents.FilterEvents[*suppliertypes.EventSupplierServicesCapped](t, sdkCtx.EventManager().Events()))
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

	_, _, err = s.keepers.ValidateSubmittedProofs(sdkCtx)
	require.NoError(t, err)

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
	for i, expectedEvent := range expectedEvents {
		require.Equal(t, int32(prooftypes.ProofRequirementReason_THRESHOLD), expectedEvent.GetProofRequirementInt())
		require.Equal(t, s.numRelays, expectedEvent.GetNumRelays())
		require.Equal(t, s.numClaimedComputeUnits, expectedEvent.GetNumClaimedComputeUnits())
		require.Equal(t, s.numEstimatedComputeUnits, expectedEvent.GetNumEstimatedComputeUnits())
		require.Equal(t, s.claimedUpokt.String(), expectedEvent.GetClaimedUpokt())

		// Validate reward distribution is not empty
		// DEV_NOTE: DOES NOT validate reward distribution amounts, this is out of scope for this test.
		rewardDistribution := expectedEvent.GetRewardDistribution()
		require.NotNil(t, rewardDistribution, "reward distribution should not be nil for event %d", i)
		require.NotEmpty(t, rewardDistribution, "reward distribution should not be empty for event %d", i)
	}
}

func (s *TestSuite) TestSettlePendingClaims_EventSettlementBatch_FieldsCorrect() {
	// Settles a single claim and validates all EventSettlementBatch event fields.
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	claim := s.claims[0]
	relayMiningDifficulty := s.relayMiningDifficulties[0]

	// Set proof params so no proof is required.
	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)
	proofRequirementThreshold = proofRequirementThreshold.Add(uPOKTCoin(1))

	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	s.keepers.UpsertClaim(ctx, claim)

	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, _, _, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), settledResults.GetNumClaims())

	events := sdkCtx.EventManager().Events()
	batchEvents := testutilevents.FilterEvents[*tokenomicstypes.EventSettlementBatch](t, events)
	require.NotEmpty(t, batchEvents, "EventSettlementBatch events should be emitted")

	// Collect the set of OpType values present.
	opTypesSeen := make(map[string]bool)
	for _, evt := range batchEvents {
		opTypesSeen[evt.OpType] = true

		// All batch events should have the correct session end height.
		require.Equal(t, sessionEndHeight, evt.SessionEndBlockHeight)

		// All batch events should have NumClaims == 1 since there's only 1 claim.
		require.Equal(t, uint32(1), evt.NumClaims)

		// OpReason should never be UNSPECIFIED.
		require.NotEqual(t, tokenomicstypes.SettlementOpReason_UNSPECIFIED, evt.OpReason)

		// TotalAmount should be a parseable non-empty coin string.
		require.NotEmpty(t, evt.TotalAmount)
		totalAmountCoin, parseErr := cosmostypes.ParseCoinNormalized(evt.TotalAmount)
		require.NoError(t, parseErr, "TotalAmount should be parseable: %s", evt.TotalAmount)
		require.True(t, totalAmountCoin.Amount.IsPositive(), "TotalAmount should be positive: %s", evt.TotalAmount)

		// Validate per-OpType fields.
		switch evt.OpType {
		case "mint", "burn":
			require.NotEmpty(t, evt.SenderModule, "%s event should have non-empty SenderModule", evt.OpType)
		case "mod_to_mod":
			require.NotEmpty(t, evt.SenderModule, "mod_to_mod event should have non-empty SenderModule")
			require.NotEmpty(t, evt.Recipient, "mod_to_mod event should have non-empty Recipient")
		case "mod_to_acct":
			require.NotEmpty(t, evt.SenderModule, "mod_to_acct event should have non-empty SenderModule")
			require.NotEmpty(t, evt.Recipient, "mod_to_acct event should have non-empty Recipient")
			// Recipient should be a valid bech32 address.
			_, addrErr := cosmostypes.AccAddressFromBech32(evt.Recipient)
			require.NoError(t, addrErr, "mod_to_acct Recipient should be valid bech32: %s", evt.Recipient)
		default:
			t.Fatalf("unexpected OpType: %s", evt.OpType)
		}
	}

	// All 4 OpType values should be present for a standard single-claim settlement.
	require.True(t, opTypesSeen["mint"], "should have mint batch events")
	require.True(t, opTypesSeen["burn"], "should have burn batch events")
	require.True(t, opTypesSeen["mod_to_mod"], "should have mod_to_mod batch events")
	require.True(t, opTypesSeen["mod_to_acct"], "should have mod_to_acct batch events")
}

func (s *TestSuite) TestSettlePendingClaims_ModuleBalances_ConsistentAfterAggregation() {
	// Tests that module balances are consistent after aggregated multi-claim settlement.
	// Setup: 3 applications, 1 supplier, 1 service (same as multi-claim test).
	// Validates: no negative intermediate balances, tokenomics module drained, conservation.
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	claim := s.claims[0]
	serviceId := claim.SessionHeader.ServiceId
	supplierOpAddr := claim.SupplierOperatorAddress

	// Set proof params so no proof is required.
	relayMiningDifficulty := s.relayMiningDifficulties[0]
	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)
	proofRequirementThreshold = proofRequirementThreshold.Add(uPOKTCoin(1))

	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Upsert first claim from setup.
	s.keepers.UpsertClaim(ctx, claim)

	// Create 2 additional applications and claims.
	for i := 0; i < 2; i++ {
		appStake := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
		appAddr := sample.AccAddressBech32()
		app := apptypes.Application{
			Address:        appAddr,
			Stake:          &appStake,
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: serviceId}},
		}
		s.keepers.SetApplication(ctx, app)

		sessionReq := &sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			ServiceId:          serviceId,
			BlockHeight:        1,
		}
		sessionRes, sessionErr := s.keepers.GetSession(sdkCtx, sessionReq)
		require.NoError(t, sessionErr)

		merkleRoot := testproof.SmstRootWithSumAndCount(5, 5)
		extraClaim := testtree.NewClaim(t, supplierOpAddr, sessionRes.Session.Header, merkleRoot)
		s.keepers.UpsertClaim(ctx, *extraClaim)
	}

	// Record pre-settlement module balances.
	queryBalance := func(moduleName string) int64 {
		res, balErr := s.keepers.Balance(ctx, &banktypes.QueryBalanceRequest{
			Address: authtypes.NewModuleAddress(moduleName).String(),
			Denom:   pocket.DenomuPOKT,
		})
		require.NoError(t, balErr)
		return res.Balance.Amount.Int64()
	}

	preAppBalance := queryBalance(apptypes.ModuleName)
	preTokenomicsBalance := queryBalance(tokenomicstypes.ModuleName)

	// Settle all 3 claims.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, _, _, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(3), settledResults.GetNumClaims())

	// Post-settlement balance assertions.
	postAppBalance := queryBalance(apptypes.ModuleName)
	postTokenomicsBalance := queryBalance(tokenomicstypes.ModuleName)
	postSupplierBalance := queryBalance(suppliertypes.ModuleName)

	// Tokenomics module should be drained (all minted funds distributed).
	require.Equal(t, preTokenomicsBalance, postTokenomicsBalance,
		"tokenomics module balance should not change (mints to other modules, not itself)")

	// Application module balance should have decreased (burns).
	require.Less(t, postAppBalance, preAppBalance,
		"application module balance should decrease from burns")

	// All module balances should be non-negative.
	require.True(t, postAppBalance >= 0, "application module balance should be >= 0")
	require.True(t, postSupplierBalance >= 0, "supplier module balance should be >= 0")
	require.True(t, postTokenomicsBalance >= 0, "tokenomics module balance should be >= 0")

	// Verify conservation via batch events: sum(mints) - sum(burns) == net supply change.
	events := sdkCtx.EventManager().Events()
	batchEvents := testutilevents.FilterEvents[*tokenomicstypes.EventSettlementBatch](t, events)

	totalMinted := math.ZeroInt()
	totalBurned := math.ZeroInt()
	for _, evt := range batchEvents {
		amountCoin, parseErr := cosmostypes.ParseCoinNormalized(evt.TotalAmount)
		require.NoError(t, parseErr)

		switch evt.OpType {
		case "mint":
			totalMinted = totalMinted.Add(amountCoin.Amount)
		case "burn":
			totalBurned = totalBurned.Add(amountCoin.Amount)
		}
	}

	// Net supply change = mints - burns.
	// This should be non-negative with the global inflation TLM.
	netSupplyChange := totalMinted.Sub(totalBurned)
	require.True(t, netSupplyChange.GTE(math.ZeroInt()),
		"net supply change should be non-negative (mints >= burns), got %s", netSupplyChange)
}

func (s *TestSuite) TestSettlePendingClaims_Overservicing_SettledUpoktReflectsCap() {
	// Tests that overservicing cap is reflected in settled_upokt for small-stake applications.
	// Setup: 2 apps — app1 with 1M stake (normal), app2 with tiny stake (overserviced).
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	claim := s.claims[0]
	serviceId := claim.SessionHeader.ServiceId
	supplierOpAddr := claim.SupplierOperatorAddress

	// Set proof params so no proof is required.
	relayMiningDifficulty := s.relayMiningDifficulties[0]
	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)
	proofRequirementThreshold = proofRequirementThreshold.Add(uPOKTCoin(1))

	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Upsert the first claim (app1 from setup with 1M stake).
	s.keepers.UpsertClaim(ctx, claim)

	// Create app2 with very small stake (100 uPOKT) — will be overserviced.
	tinyStake := cosmostypes.NewCoin("upokt", math.NewInt(100))
	tinyAppAddr := sample.AccAddressBech32()
	tinyApp := apptypes.Application{
		Address:        tinyAppAddr,
		Stake:          &tinyStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: serviceId}},
	}
	s.keepers.SetApplication(ctx, tinyApp)

	sessionReq := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: tinyAppAddr,
		ServiceId:          serviceId,
		BlockHeight:        1,
	}
	sessionRes, sessionErr := s.keepers.GetSession(sdkCtx, sessionReq)
	require.NoError(t, sessionErr)

	// Create a claim for app2 with 5 relays (same as app1).
	merkleRoot := testproof.SmstRootWithSumAndCount(5, 5)
	tinyAppClaim := testtree.NewClaim(t, supplierOpAddr, sessionRes.Session.Header, merkleRoot)
	s.keepers.UpsertClaim(ctx, *tinyAppClaim)

	// Settle both claims.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, _, _, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), settledResults.GetNumClaims())

	events := sdkCtx.EventManager().Events()

	// Check for EventApplicationOverserviced — should be emitted for app2 only.
	overservicedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventApplicationOverserviced](t, events)
	require.Equal(t, 1, len(overservicedEvents), "exactly one overserviced event for tiny app")
	require.Equal(t, tinyAppAddr, overservicedEvents[0].ApplicationAddr)

	// Validate EventClaimSettled events.
	claimSettledEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events)
	require.Equal(t, 2, len(claimSettledEvents))

	// Find the settled event for the tiny app (app2) and for the normal app (app1).
	for _, evt := range claimSettledEvents {
		if evt.ApplicationAddress == tinyAppAddr {
			// For the overserviced app: settled_upokt should be less than claimed_upokt.
			require.NotEmpty(t, evt.SettledUpokt, "settled_upokt should be populated for tiny app")
			settledCoin, parseErr := cosmostypes.ParseCoinNormalized(evt.SettledUpokt)
			require.NoError(t, parseErr)
			claimedCoin, parseErr := cosmostypes.ParseCoinNormalized(evt.ClaimedUpokt)
			require.NoError(t, parseErr)
			require.True(t, settledCoin.Amount.LT(claimedCoin.Amount),
				"tiny app settled_upokt (%s) should be less than claimed_upokt (%s)",
				settledCoin, claimedCoin)
		} else {
			// For the normal app: settled_upokt should be populated and reasonable.
			require.NotEmpty(t, evt.SettledUpokt, "settled_upokt should be populated for normal app")
		}
	}

	// Application module balance should be non-negative.
	appModuleBalance, err := s.keepers.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: authtypes.NewModuleAddress(apptypes.ModuleName).String(),
		Denom:   pocket.DenomuPOKT,
	})
	require.NoError(t, err)
	require.True(t, appModuleBalance.Balance.Amount.GTE(math.ZeroInt()),
		"application module balance should be >= 0 after overserviced settlement")
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

// TestSettlePendingClaims_NoClaims_NoPanic verifies that the two-pass settlement
// handles zero claims gracefully: no error, no panic, no events emitted.
func (s *TestSuite) TestSettlePendingClaims_NoClaims_NoPanic() {
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Do NOT insert any claims into the store.

	// Use one of the pre-created claims only to derive a valid settlement height
	// (proof window close height), then settle at that height with an empty store.
	sessionEndHeight := s.claims[0].SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)

	// Settlement must not panic and must return no error.
	settledResults, expiredResults, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// No claims should have been settled, expired, or discarded.
	require.Equal(t, uint64(0), settledResults.GetNumClaims())
	require.Equal(t, uint64(0), expiredResults.GetNumClaims())
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)

	// No claims should remain in the store.
	claims := s.keepers.GetAllClaims(ctx)
	require.Equal(t, 0, len(claims))

	// No settlement events should have been emitted.
	events := sdkCtx.EventManager().Events()
	settlementEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events)
	require.Equal(t, 0, len(settlementEvents))
}

// TestSettlePendingClaims_CuttmChangeMidFlight_PaysAtSessionStartRate pins that a claim is
// settled using the compute_units_to_tokens_multiplier that was effective at its SESSION
// START, not the live value at the settlement block.
//
// Regression: settlement used to read live shared params for pricing, while x/proof priced
// the same claim (and decided whether a proof was required) at the session start height. A
// governance CUTTM change landing between a session's start and its settlement therefore
// paid the supplier at a rate the claim was never priced or proof-gated under. The
// divergence was visible inside settleClaim itself, which derived claimeduPOKT from live
// params and then called ProofRequirementForClaim, which re-derived it at session start.
//
// Without the session-start pin this test fails with claimed_upokt at 2x the expected value.
func (s *TestSuite) TestSettlePendingClaims_CuttmChangeMidFlight_PaysAtSessionStartRate() {
	t := s.T()
	ctx := s.ctx

	// Genesis params — the epoch every claim from SetupTest was created under (session
	// start height 1). s.claimedUpokt was computed with this CUTTM.
	genesisParams := s.keepers.SharedKeeper.GetParams(ctx)

	claim := s.claims[0]
	relayMiningDifficulty := s.relayMiningDifficulties[0]

	// Set the proof params so this claim does NOT require a proof, using the OLD rate:
	// ProofRequirementForClaim resolves CUTTM at session start, so the threshold must be
	// compared against the session-start price to keep this test about pricing only.
	proofRequirementThreshold, err := claim.GetClaimeduPOKT(genesisParams, relayMiningDifficulty)
	require.NoError(t, err)
	proofRequirementThreshold = proofRequirementThreshold.Add(uPOKTCoin(1))

	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	require.NoError(t, s.keepers.ProofKeeper.SetParams(ctx, proofParams))

	// Record the genesis epoch in params history at height 1, mirroring what
	// recordParamsHistory seeds on the first-ever shared param update. This is what makes
	// GetParamsAtHeight(sessionStartHeight=1) resolve to the OLD CUTTM.
	historicalParams := genesisParams
	historicalParams.SessionGridAnchorHeight = 1
	historicalParams.SessionNumberAtAnchor = 1
	// SetParamsAtHeight is on the concrete shared keeper; the tokenomics SharedKeeper
	// interface intentionally exposes reads only.
	concreteSharedKeeper, ok := s.keepers.SharedKeeper.(*sharedkeeper.Keeper)
	require.True(t, ok, "expected the test harness to hold a concrete *sharedkeeper.Keeper")
	require.NoError(t, concreteSharedKeeper.SetParamsAtHeight(ctx, 1, historicalParams))

	// Now double the LIVE CUTTM, simulating a governance change promoted after the session
	// started but before it settles. Only the multiplier changes — no session-timing param
	// is touched, so the session grid (and therefore the settlement height below) is
	// unaffected and this test isolates pricing.
	liveParams := genesisParams
	liveParams.ComputeUnitsToTokensMultiplier = genesisParams.ComputeUnitsToTokensMultiplier * 2
	require.NoError(t, s.keepers.SharedKeeper.SetParams(ctx, liveParams))

	// Sanity check: live and session-start params must actually disagree, otherwise this
	// test would pass trivially even with the bug present.
	require.NotEqual(t,
		s.keepers.SharedKeeper.GetParams(ctx).ComputeUnitsToTokensMultiplier,
		s.keepers.SharedKeeper.GetParamsAtHeight(ctx, 1).ComputeUnitsToTokensMultiplier,
		"live and session-start CUTTM must differ for this regression to be meaningful",
	)

	s.keepers.UpsertClaim(ctx, claim)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	s.keepers.ValidateSubmittedProofs(sdkCtx)

	// Settle after the proof window closes. Timing params are unchanged between the two
	// epochs, so either one yields the same close height.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&genesisParams, sessionEndHeight)
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)

	settledResults, expiredResults, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), settledResults.GetNumClaims())
	require.Equal(t, uint64(0), expiredResults.GetNumClaims())
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)

	// The settled amount must match the session-start rate, NOT the doubled live rate.
	events := sdkCtx.EventManager().Events()
	settlementEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events)
	require.Equal(t, 1, len(settlementEvents))

	settlementEvent := settlementEvents[0]
	require.Equal(t, int32(prooftypes.ProofRequirementReason_NOT_REQUIRED), settlementEvent.GetProofRequirementInt())
	require.Equal(t, s.claimedUpokt.String(), settlementEvent.GetClaimedUpokt(),
		"claim must settle at the session-start CUTTM, not the live (doubled) one",
	)
}

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
			OwnerAddress:         sample.AccAddressBech32(),
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

		// Construct a valid session tree with 5 relays.
		s.numRelays = uint64(5)
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

// TestSettlePendingClaims_GetActualSupplierCount_Fallback verifies that
// GetActualSupplierCount returns 1 (safe fallback) when IncrementSupplierCount
// was never called for a given (app, session) pair.
func (s *TestSuite) TestSettlePendingClaims_GetActualSupplierCount_Fallback() {
	t := s.T()
	ctx := s.ctx

	sctx := tokenomicskeeper.NewSettlementContext(ctx, s.keepers.Keeper, s.keepers.Logger())

	// Query a supplier count for a pair that was never incremented.
	count := sctx.GetActualSupplierCount("nonexistent-app", "nonexistent-session")
	require.Equal(t, int64(1), count, "fallback should return 1 for unknown (app, session) pair")

	// Verify normal increment works and returns correct count.
	sctx.IncrementSupplierCount("app1", "session1")
	sctx.IncrementSupplierCount("app1", "session1")
	sctx.IncrementSupplierCount("app1", "session1")
	count = sctx.GetActualSupplierCount("app1", "session1")
	require.Equal(t, int64(3), count, "should return 3 after 3 increments")

	// Verify different (app, session) pairs are independent.
	sctx.IncrementSupplierCount("app2", "session1")
	count = sctx.GetActualSupplierCount("app2", "session1")
	require.Equal(t, int64(1), count, "different app should have independent count")
}

// TestSettlePendingClaims_FlushBatchedValidatorRewards_EmptyAccumulator verifies that
// FlushBatchedValidatorRewards returns (nil, nil) when the validator reward accumulator
// is empty — i.e., no rewards were added to the settlement context.
func (s *TestSuite) TestSettlePendingClaims_FlushBatchedValidatorRewards_EmptyAccumulator() {
	t := s.T()
	ctx := s.ctx

	// Create a fresh settlement context without adding any validator rewards.
	sctx := tokenomicskeeper.NewSettlementContext(ctx, s.keepers.Keeper, s.keepers.Logger())

	// Flush with an empty accumulator should return (nil, nil).
	result, err := s.keepers.FlushBatchedValidatorRewards(ctx, sctx, int64(0))
	require.NoError(t, err, "flushing an empty accumulator should not return an error")
	require.Nil(t, result, "flushing an empty accumulator should return a nil result")
}

// A claim for a session the supplier served settles even when the claimed service
// is deactivated before the settlement height.
func (s *TestSuite) TestSettlePendingClaims_ServiceDeactivatedAfterSession_StillSettles() {
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	claim := s.claims[0]
	relayMiningDifficulty := s.relayMiningDifficulties[0]

	// No proof required.
	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)
	proofRequirementThreshold = proofRequirementThreshold.Add(uPOKTCoin(1))
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	require.NoError(t, s.keepers.ProofKeeper.SetParams(ctx, proofParams))

	s.keepers.UpsertClaim(ctx, claim)

	// Deactivate the claimed service from the session after the claimed one.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	supplier, found := s.keepers.GetSupplier(ctx, claim.SupplierOperatorAddress)
	require.True(t, found)
	for _, config := range supplier.ServiceConfigHistory {
		if config.Service.ServiceId == claim.SessionHeader.ServiceId {
			config.DeactivationHeight = sessionEndHeight + 1
		}
	}
	s.keepers.SetAndIndexDehydratedSupplier(ctx, supplier)

	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, expiredResults, numDiscarded, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), settledResults.GetNumClaims())
	require.Equal(t, uint64(0), expiredResults.GetNumClaims())
	require.Equal(t, uint64(0), numDiscarded)
}

// A claim for a service the service cap deactivated during the claimed session
// still settles: the supplier served that session.
func (s *TestSuite) TestSettlePendingClaims_ServiceCappedDuringSession_StillSettles() {
	t := s.T()
	price := supplierStakeAmt / 10
	settleHeight, _, supplierOperatorAddr := s.setupSlashCapScenario(price, []string{"svc_extra_1", "svc_extra_2"}, true)

	// The stake covers only 2 of the 3 services.
	supplierKeeper := s.keepers.SupplierKeeper.(*supplierkeeper.Keeper)
	supplierParams := supplierKeeper.GetParams(s.ctx)
	minStake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, supplierStakeAmt-price)
	supplierParams.MinStake = &minStake
	require.NoError(t, supplierKeeper.SetParams(s.ctx, supplierParams))

	// A claim for svc_extra_2 in the first session, no proof required.
	s.upsertMissingProofClaim("svc_extra_2", supplierOperatorAddr)
	proofParams := s.keepers.ProofKeeper.GetParams(s.ctx)
	huge := uPOKTCoin(1_000_000_000_000)
	proofParams.ProofRequirementThreshold = &huge
	proofParams.ProofRequestProbability = 0
	require.NoError(t, s.keepers.ProofKeeper.SetParams(s.ctx, proofParams))

	// The cap is enforced mid-session.
	capCtx := cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(5)
	supplier, found := supplierKeeper.GetDehydratedSupplier(capCtx, supplierOperatorAddr)
	require.True(t, found)
	deactivated := supplierKeeper.EnforceServiceCap(capCtx, &supplier, suppliertypes.ServiceCapReason_SERVICE_CAP_REASON_PARAMS)
	require.Equal(t, []string{"svc_extra_2"}, deactivated)

	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(settleHeight)
	settledResults, _, numDiscarded, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), settledResults.GetNumClaims())
	require.Equal(t, uint64(0), numDiscarded)
}

// If the config active at the claim's session start was already pruned, settlement
// falls back to the supplier's config for that service active now.
func (s *TestSuite) TestSettlePendingClaims_PrunedSessionConfig_FallsBackToCurrent() {
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	claim := s.claims[0]
	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulties[0])
	require.NoError(t, err)
	proofRequirementThreshold = proofRequirementThreshold.Add(uPOKTCoin(1))
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	require.NoError(t, s.keepers.ProofKeeper.SetParams(ctx, proofParams))
	s.keepers.UpsertClaim(ctx, claim)

	// Replace the claimed service's config with one active only from the next
	// session on, dropping the old config from the history (as pruning does).
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	supplier, found := s.keepers.GetSupplier(ctx, claim.SupplierOperatorAddress)
	require.True(t, found)
	history := make([]*sharedtypes.ServiceConfigUpdate, 0, len(supplier.ServiceConfigHistory))
	for _, config := range supplier.ServiceConfigHistory {
		if config.Service.ServiceId == claim.SessionHeader.ServiceId {
			replacement := *config
			replacement.ActivationHeight = sessionEndHeight + 1
			replacement.DeactivationHeight = sharedtypes.NoDeactivationHeight
			config = &replacement
		}
		history = append(history, config)
	}
	supplier.ServiceConfigHistory = history
	s.keepers.SetAndIndexDehydratedSupplier(ctx, supplier)

	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, _, numDiscarded, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), settledResults.GetNumClaims())
	require.Equal(t, uint64(0), numDiscarded)
}

// Claims of one supplier and service from different sessions settle with the
// revenue share active at each claim's own session start, in either order.
func (s *TestSuite) TestSettlementContext_SupplierRevShareResolvedPerSession() {
	t := s.T()
	ctx := cosmostypes.UnwrapSDKContext(s.ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	claimOld := s.claims[0]
	serviceId := claimOld.SessionHeader.ServiceId
	supplierAddr := claimOld.SupplierOperatorAddress
	oldStart := claimOld.SessionHeader.SessionStartBlockHeight
	newStart := sharedtypes.GetNextSessionStartHeight(&sharedParams, claimOld.SessionHeader.SessionEndBlockHeight)

	supplier, found := s.keepers.GetSupplier(ctx, supplierAddr)
	require.True(t, found)

	// Change the service's revenue share from the next session on.
	newRevShareAddr := sample.AccAddressBech32()
	history := make([]*sharedtypes.ServiceConfigUpdate, 0)
	for _, cfg := range supplier.ServiceConfigHistory {
		history = append(history, cfg)
		if cfg.Service.ServiceId != serviceId {
			continue
		}
		cfg.DeactivationHeight = newStart
		history = append(history, &sharedtypes.ServiceConfigUpdate{
			OperatorAddress: supplierAddr,
			Service: &sharedtypes.SupplierServiceConfig{
				ServiceId: serviceId,
				RevShare:  []*sharedtypes.ServiceRevenueShare{{Address: newRevShareAddr, RevSharePercentage: 100}},
			},
			ActivationHeight: newStart,
		})
	}
	supplier.ServiceConfigHistory = history
	s.keepers.SetAndIndexDehydratedSupplier(ctx, supplier)

	claimNew := claimOld
	headerNew := *claimOld.SessionHeader
	headerNew.SessionStartBlockHeight = newStart
	headerNew.SessionEndBlockHeight = sharedtypes.GetSessionEndHeight(&sharedParams, newStart)
	headerNew.SessionId = "next_session"
	claimNew.SessionHeader = &headerNew

	settleCtx := ctx.WithBlockHeight(headerNew.SessionEndBlockHeight + 1)
	revShareAddr := func(sctx interface {
		GetSupplierForClaim(string, string, int64) (*sharedtypes.Supplier, error)
	}, sessionStart int64) string {
		claimSupplier, err := sctx.GetSupplierForClaim(supplierAddr, serviceId, sessionStart)
		require.NoError(t, err)
		require.Len(t, claimSupplier.Services, 1)
		return claimSupplier.Services[0].RevShare[0].Address
	}

	for _, claims := range [][]*prooftypes.Claim{{&claimNew, &claimOld}, {&claimOld, &claimNew}} {
		sctx := tokenomicskeeper.NewSettlementContext(settleCtx, s.keepers.Keeper, s.keepers.Logger())
		for _, claim := range claims {
			require.NoError(t, sctx.ClaimCacheWarmUp(settleCtx, claim))
		}
		require.Equal(t, supplier.OwnerAddress, revShareAddr(sctx, oldStart))
		require.Equal(t, newRevShareAddr, revShareAddr(sctx, newStart))
	}
}

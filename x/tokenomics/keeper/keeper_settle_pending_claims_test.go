package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	testutilproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testtree"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/tokenomics"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	testServiceId = "svc1"
	supplierStake = 1000000 // uPOKT
)

func init() {
	cmd.InitSDKConfig()
}

// TODO_TECHDEBT(@olshansk): Consolidate the setup for all tests that use TokenomicsModuleKeepers
type TestSuite struct {
	suite.Suite

	ctx     context.Context
	keepers keepertest.TokenomicsModuleKeepers
	claim   prooftypes.Claim
	proof   prooftypes.Proof

	numRelays uint64
}

// SetupTest creates the following and stores them in the suite:
// - An cosmostypes.Context.
// - A keepertest.TokenomicsModuleKeepers to provide access to integrated keepers.
// - An expectedComputeUnits which is the default proof_requirement_threshold.
// - A claim that will require a proof via threshold, given the default proof params.
// - A proof which contains only the session header supplier operator address.
func (s *TestSuite) SetupTest() {
	t := s.T()

	s.keepers, s.ctx = keepertest.NewTokenomicsModuleKeepers(s.T(), nil)
	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(1)

	// Add a block proposer address to the context
	valAddr, err := cosmostypes.ValAddressFromBech32(sample.ConsAddress())
	require.NoError(t, err)
	consensusAddr := cosmostypes.ConsAddress(valAddr)
	sdkCtx = sdkCtx.WithProposer(consensusAddr)

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(s.keepers.Codec)

	// Create a pre-generated account iterator to create accounts for the test.
	preGeneratedAccts := testkeyring.PreGeneratedAccounts()

	// Create accounts in the account keeper with corresponding keys in the keyring
	// // for the applications and suppliers used in the tests.
	supplierOwnerAddr := testkeyring.CreateOnChainAccount(
		sdkCtx, t,
		"supplier",
		keyRing,
		s.keepers.AccountKeeper,
		preGeneratedAccts,
	).String()
	appAddr := testkeyring.CreateOnChainAccount(
		sdkCtx, t,
		"app",
		keyRing,
		s.keepers.AccountKeeper,
		preGeneratedAccts,
	).String()

	service := sharedtypes.Service{
		Id:                   testServiceId,
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         sample.AccAddress(),
	}
	s.keepers.SetService(s.ctx, service)

	supplierStake := types.NewCoin("upokt", math.NewInt(supplierStake))
	supplier := sharedtypes.Supplier{
		OwnerAddress:    supplierOwnerAddr,
		OperatorAddress: supplierOwnerAddr,
		Stake:           &supplierStake,
		Services: []*sharedtypes.SupplierServiceConfig{{
			ServiceId: testServiceId,
			RevShare: []*sharedtypes.ServiceRevenueShare{{
				Address:            supplierOwnerAddr,
				RevSharePercentage: 100,
			}},
		}},
	}
	s.keepers.SetSupplier(s.ctx, supplier)

	appStake := types.NewCoin("upokt", math.NewInt(1000000))
	app := apptypes.Application{
		Address:        appAddr,
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: testServiceId}},
	}
	s.keepers.SetApplication(s.ctx, app)

	// Get the session for the application/supplier pair which is expected
	// to be claimed and for which a valid proof would be accepted.
	sessionReq := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddr,
		ServiceId:          testServiceId,
		BlockHeight:        1,
	}
	sessionRes, err := s.keepers.GetSession(sdkCtx, sessionReq)
	require.NoError(t, err)
	sessionHeader := sessionRes.Session.Header

	// Construct a ringClient to get the application's ring & verify the relay
	// request signature.
	ringClient, err := rings.NewRingClient(depinject.Supply(
		polyzero.NewLogger(),
		prooftypes.NewAppKeeperQueryClient(s.keepers.ApplicationKeeper),
		prooftypes.NewAccountKeeperQueryClient(s.keepers.AccountKeeper),
		prooftypes.NewSharedKeeperQueryClient(s.keepers.SharedKeeper, s.keepers.SessionKeeper),
	))
	require.NoError(t, err)

	// Construct a valid session tree with 100 relays.
	s.numRelays = uint64(100)
	sessionTree := testtree.NewFilledSessionTree(
		sdkCtx, t,
		s.numRelays, service.ComputeUnitsPerRelay,
		"supplier", supplierOwnerAddr,
		sessionHeader, sessionHeader, sessionHeader,
		keyRing,
		ringClient,
	)

	blockHeaderHash := make([]byte, 0)
	expectedMerkleProofPath := protocol.GetPathForProof(blockHeaderHash, sessionHeader.SessionId)

	// Advance the block height to the earliest claim commit height.
	sharedParams := s.keepers.SharedKeeper.GetParams(sdkCtx)
	claimMsgHeight := shared.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
		blockHeaderHash,
		supplierOwnerAddr,
	)
	sdkCtx = sdkCtx.WithBlockHeight(claimMsgHeight).WithHeaderHash(blockHeaderHash)
	s.ctx = sdkCtx

	merkleRootBz, err := sessionTree.Flush()
	require.NoError(t, err)

	// Prepare a claim that can be inserted
	s.claim = *testtree.NewClaim(t, supplierOwnerAddr, sessionHeader, merkleRootBz)
	s.proof = *testtree.NewProof(t, supplierOwnerAddr, sessionHeader, sessionTree, expectedMerkleProofPath)
}

// TestSettleExpiringClaimsSuite tests the claim settlement process.
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

	// Upsert the claim only
	s.keepers.UpsertClaim(ctx, s.claim)

	// Settle pending claims while the session is still active.
	// Expectations: No claims should be settled because the session is still ongoing
	blockHeight := s.claim.SessionHeader.SessionEndBlockHeight - 2 // session is still active
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled or expired.
	require.Equal(t, uint64(0), settledResult.NumClaims)
	require.Equal(t, uint64(0), expiredResult.NumClaims)

	// Validate that one claim still remains.
	claims := s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)

	// Calculate a block height which is within the proof window.
	proofWindowOpenHeight := shared.GetProofWindowOpenHeight(
		&sharedParams, s.claim.SessionHeader.SessionEndBlockHeight,
	)
	proofWindowCloseHeight := shared.GetProofWindowCloseHeight(
		&sharedParams, s.claim.SessionHeader.SessionEndBlockHeight,
	)
	blockHeight = (proofWindowCloseHeight - proofWindowOpenHeight) / 2

	// 2. Settle pending claims just after the session ended.
	// Expectations: Claims should not be settled because the proof window hasn't closed yet.
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResult, expiredResult, err = s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled or expired.
	require.Equal(t, uint64(0), settledResult.NumClaims)
	require.Equal(t, uint64(0), expiredResult.NumClaims)

	// Validate that the claim still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)
}

func (s *TestSuite) TestSettlePendingClaims_ClaimExpired_ProofRequiredAndNotProvided_ViaThreshold() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Retrieve the number of compute units in the claim
	numComputeUnits, err := s.claim.GetNumComputeUnits()
	require.NoError(t, err)

	tokenomicsParams := s.keepers.Keeper.GetParams(ctx)
	// -1 to push threshold below s.claim's compute units
	proofRequirementThreshold, err := tokenomics.NumComputeUnitsToCoin(tokenomicsParams, numComputeUnits-1)
	require.NoError(t, err)

	// Set the proof missing penalty to half the supplier's stake so it is not
	// unstaked when being slashed.
	belowStakeAmountProofMissingPenalty := sdk.NewCoin(volatile.DenomuPOKT, math.NewInt(supplierStake/2))

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
	s.keepers.UpsertClaim(ctx, s.claim)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be expired.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := s.claim.SessionHeader.SessionEndBlockHeight
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(0), settledResult.NumClaims) // 0 claims settled
	require.Equal(t, uint64(1), expiredResult.NumClaims) // 1 claim expired

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Slashing should have occurred without unstaking the supplier.
	// The supplier is not unstaked because it got slashed by an amount that is
	// half its stake (i.e. missing proof penalty == stake / 2), resulting in a
	// remaining stake that is above the minimum stake (i.e. new_stake == prev_stake / 2).
	slashedSupplier, supplierFound := s.keepers.GetSupplier(sdkCtx, s.claim.SupplierOperatorAddress)
	require.True(t, supplierFound)
	require.Equal(t, math.NewInt(supplierStake/2), slashedSupplier.Stake.Amount)
	require.Equal(t, uint64(0), slashedSupplier.UnstakeSessionEndHeight)

	events := sdkCtx.EventManager().Events()
	require.Len(t, events, 10) // asserting on the length of events so the developer must consciously update it upon changes

	// Confirm an expiration event was emitted
	expectedClaimExpiredEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimExpired](t, events, "poktroll.tokenomics.EventClaimExpired")
	require.Len(t, expectedClaimExpiredEvents, 1)

	// Validate the claim expired event
	expectedClaimExpiredEvent := expectedClaimExpiredEvents[0]
	require.Equal(t, tokenomicstypes.ClaimExpirationReason_PROOF_MISSING, expectedClaimExpiredEvent.GetExpirationReason())
	require.Equal(t, s.numRelays, expectedClaimExpiredEvent.GetNumRelays())

	// Confirm that a slashing event was emitted
	expectedSlashingEvents := testutilevents.FilterEvents[*tokenomicstypes.EventSupplierSlashed](t, events, "poktroll.tokenomics.EventSupplierSlashed")
	require.Len(t, expectedSlashingEvents, 1)

	// Validate the slashing event
	expectedSlashingEvent := expectedSlashingEvents[0]
	require.Equal(t, slashedSupplier.GetOperatorAddress(), expectedSlashingEvent.GetSupplierOperatorAddr())
	require.Equal(t, uint64(1), expectedSlashingEvent.GetNumExpiredClaims())
	require.Equal(t, &belowStakeAmountProofMissingPenalty, expectedSlashingEvent.GetSlashingAmount())
}

func (s *TestSuite) TestSettlePendingClaims_ClaimSettled_ProofRequiredAndProvided_ViaThreshold() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Retrieve the number of compute units in the claim
	numComputeUnits, err := s.claim.GetNumComputeUnits()
	require.NoError(t, err)

	tokenomicsParams := s.keepers.Keeper.GetParams(ctx)
	// -1 to push threshold below s.claim's compute units
	proofRequirementThreshold, err := tokenomics.NumComputeUnitsToCoin(tokenomicsParams, numComputeUnits-1)
	require.NoError(t, err)

	// Set the proof parameters such that s.claim requires a proof because:
	// - proof_request_probability is 0%
	// - proof_requirement_threshold is below the claim (i.e. claim is above threshold)
	err = s.keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability:   0,
		ProofRequirementThreshold: &proofRequirementThreshold,
	})
	require.NoError(t, err)

	// Upsert the claim & proof
	s.keepers.UpsertClaim(ctx, s.claim)
	s.keepers.UpsertProof(ctx, s.proof)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := s.claim.SessionHeader.SessionEndBlockHeight
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(1), settledResult.NumClaims) // 1 claim settled
	require.Equal(t, uint64(0), expiredResult.NumClaims) // 0 claims expired

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an settlement event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events, "poktroll.tokenomics.EventClaimSettled")
	require.Len(t, expectedEvents, 1)

	// Validate the event
	expectedEvent := expectedEvents[0]
	require.Equal(t, prooftypes.ProofRequirementReason_THRESHOLD, expectedEvent.GetProofRequirement())
	require.Equal(t, s.numRelays, expectedEvent.GetNumRelays())
}

func (s *TestSuite) TestSettlePendingClaims_ClaimExpired_ProofRequired_InvalidOneProvided() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	belowStakeAmountProofMissingPenalty := sdk.NewCoin(volatile.DenomuPOKT, math.NewInt(supplierStake/2))

	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	// Set the proof parameters such that s.claim DOES NOT require a proof because:
	// - proof_request_probability is 100%
	proofParams.ProofRequestProbability = 1
	// Set the proof missing penalty to half the supplier's stake so it is not
	// unstaked when being slashed.
	proofParams.ProofMissingPenalty = &belowStakeAmountProofMissingPenalty
	err := s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Create a claim that requires a proof and an invalid proof
	proof := s.proof
	proof.ClosestMerkleProof = []byte("invalid_proof")

	// Upsert the proof & claim
	s.keepers.UpsertClaim(ctx, s.claim)
	s.keepers.UpsertProof(ctx, proof)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be expired.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := s.claim.SessionHeader.SessionEndBlockHeight
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(0), settledResult.NumClaims) // 0 claims settled
	require.Equal(t, uint64(1), expiredResult.NumClaims) // 1 claim expired

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Validate that no proofs remain.
	proofs := s.keepers.GetAllProofs(ctx)
	require.Len(t, proofs, 0)

	// Slashing should have occurred without unstaking the supplier.
	slashedSupplier, supplierFound := s.keepers.GetSupplier(sdkCtx, s.claim.SupplierOperatorAddress)
	require.True(t, supplierFound)
	require.Equal(t, math.NewInt(supplierStake/2), slashedSupplier.Stake.Amount)
	require.Equal(t, uint64(0), slashedSupplier.UnstakeSessionEndHeight)

	// Confirm an expiration event was emitted
	events := sdkCtx.EventManager().Events()
	require.Len(t, events, 10) // minting, burning, settling, etc..
	expectedClaimExpiredEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimExpired](t, events, "poktroll.tokenomics.EventClaimExpired")
	require.Len(t, expectedClaimExpiredEvents, 1)

	// Validate the event
	expectedClaimExpiredEvent := expectedClaimExpiredEvents[0]
	require.Equal(t, tokenomicstypes.ClaimExpirationReason_PROOF_INVALID, expectedClaimExpiredEvent.GetExpirationReason())
	require.Equal(t, s.numRelays, expectedClaimExpiredEvent.GetNumRelays())

	// Confirm that a slashing event was emitted
	expectedSlashingEvents := testutilevents.FilterEvents[*tokenomicstypes.EventSupplierSlashed](t, events, "poktroll.tokenomics.EventSupplierSlashed")
	require.Len(t, expectedSlashingEvents, 1)

	// Validate the slashing event
	expectedSlashingEvent := expectedSlashingEvents[0]
	require.Equal(t, slashedSupplier.GetOperatorAddress(), expectedSlashingEvent.GetSupplierOperatorAddr())
	require.Equal(t, uint64(1), expectedSlashingEvent.GetNumExpiredClaims())
	require.Equal(t, &belowStakeAmountProofMissingPenalty, expectedSlashingEvent.GetSlashingAmount())
}

func (s *TestSuite) TestClaimSettlement_ClaimSettled_ProofRequiredAndProvided_ViaProbability() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Retrieve the number of compute units in the claim
	numComputeUnits, err := s.claim.GetNumComputeUnits()
	require.NoError(t, err)

	tokenomicsParams := s.keepers.Keeper.GetParams(ctx)
	// +1 so its not required via probability
	proofRequirementThreshold, err := tokenomics.NumComputeUnitsToCoin(tokenomicsParams, numComputeUnits+1)
	require.NoError(t, err)

	// Set the proof parameters such that s.claim requires a proof because:
	// - proof_request_probability is 100%
	// - proof_requirement_threshold is 0, should not matter
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 1
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Upsert the claim & proof
	s.keepers.UpsertClaim(ctx, s.claim)
	s.keepers.UpsertProof(ctx, s.proof)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proof window has definitely closed at this point
	sessionEndHeight := s.claim.SessionHeader.SessionEndBlockHeight
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(1), settledResult.NumClaims) // 1 claim settled
	require.Equal(t, uint64(0), expiredResult.NumClaims) // 0 claims expired

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an settlement event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events, "poktroll.tokenomics.EventClaimSettled")
	require.Len(t, expectedEvents, 1)

	// Validate the settlement event
	expectedEvent := expectedEvents[0]
	require.Equal(t, prooftypes.ProofRequirementReason_PROBABILISTIC, expectedEvent.GetProofRequirement())
	require.Equal(t, s.numRelays, expectedEvent.GetNumRelays())
}

func (s *TestSuite) TestSettlePendingClaims_Settles_WhenAProofIsNotRequired() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Retrieve the number of compute units in the claim
	numComputeUnits, err := s.claim.GetNumComputeUnits()
	require.NoError(t, err)

	tokenomicsParams := s.keepers.Keeper.GetParams(ctx)
	// +1 to push threshold above s.claim's compute units
	proofRequirementThreshold, err := tokenomics.NumComputeUnitsToCoin(tokenomicsParams, numComputeUnits+1)
	require.NoError(t, err)

	// Set the proof parameters such that s.claim DOES NOT require a proof because:
	// - proof_request_probability is 0% AND
	// - proof_requirement_threshold exceeds s.claim's compute units
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Upsert the claim only (not the proof)
	s.keepers.UpsertClaim(ctx, s.claim)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := s.claim.SessionHeader.SessionEndBlockHeight
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that one claim was settled.
	require.Equal(t, uint64(1), settledResult.NumClaims) // 1 claim settled
	require.Equal(t, uint64(0), expiredResult.NumClaims) // 0 claims expired

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm a settlement event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events, "poktroll.tokenomics.EventClaimSettled")
	require.Len(t, expectedEvents, 1)

	// Validate the settlement event
	expectedEvent := expectedEvents[0]
	require.Equal(t, prooftypes.ProofRequirementReason_NOT_REQUIRED.String(), expectedEvent.GetProofRequirement().String())
	require.Equal(t, s.numRelays, expectedEvent.GetNumRelays())
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

	// Retrieve the number of compute units in the claim
	numComputeUnits, err := s.claim.GetNumComputeUnits()
	require.NoError(t, err)

	tokenomicsParams := s.keepers.Keeper.GetParams(ctx)
	// +1 to push threshold above s.claim's compute units
	proofRequirementThreshold, err := tokenomics.NumComputeUnitsToCoin(tokenomicsParams, numComputeUnits+1)
	require.NoError(t, err)

	// Set the proof parameters such that s.claim DOES NOT require a proof
	// because the proof_request_probability is 0% and the proof_request_threshold
	// is greater than the claims' compute units.
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// 0. Add the claims & verify they exists
	sessionOneClaim := s.claim
	s.keepers.UpsertClaim(ctx, sessionOneClaim)

	sessionOneEndHeight := sessionOneClaim.GetSessionHeader().GetSessionEndBlockHeight()

	// Add a second claim with a session header corresponding to the next session.
	sessionTwoClaim := testutilproof.BaseClaim(
		sessionOneClaim.GetSessionHeader().GetServiceId(),
		sessionOneClaim.GetSessionHeader().GetApplicationAddress(),
		sessionOneClaim.GetSupplierOperatorAddress(),
		s.numRelays,
	)

	sessionOneProofWindowCloseHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionOneEndHeight)
	sessionTwoStartHeight := shared.GetSessionStartHeight(&sharedParams, sessionOneProofWindowCloseHeight+1)
	sessionTwoProofWindowCloseHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionTwoStartHeight)

	sessionTwoClaim.SessionHeader = &sessiontypes.SessionHeader{
		ApplicationAddress:      sessionOneClaim.GetSessionHeader().GetApplicationAddress(),
		ServiceId:               s.claim.GetSessionHeader().GetServiceId(),
		SessionId:               "session_two_id",
		SessionStartBlockHeight: sessionTwoStartHeight,
		SessionEndBlockHeight:   shared.GetSessionEndHeight(&sharedParams, sessionTwoStartHeight),
	}
	s.keepers.UpsertClaim(ctx, sessionTwoClaim)

	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Equalf(2, len(claims), "expected %d claims, got %d", 2, len(claims))

	// 1. Settle pending claims while the session is still active.
	// Expectations: No claims should be settled because the session is still ongoing
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionOneEndHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that one claim was settled.
	require.Equal(t, uint64(1), settledResult.NumClaims)

	// Validate that no claims expired.
	require.Equal(t, uint64(0), expiredResult.NumClaims)

	// Validate that one claim still remains.
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)

	// Calculate a block height which is within session two's proof window.
	blockHeight = (sessionTwoProofWindowCloseHeight - sessionTwoStartHeight) / 2

	// 2. Settle pending claims just after the session ended.
	// Expectations: Claims should not be settled because the proof window hasn't closed yet.
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResult, expiredResult, err = s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled or expired.
	require.Equal(t, uint64(0), settledResult.NumClaims)
	require.Equal(t, uint64(0), expiredResult.NumClaims)

	// Validate that the claim still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)
}

func (s *TestSuite) TestSettlePendingClaims_ClaimExpired_SupplierUnstaked() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Retrieve the number of compute units in the claim
	numComputeUnits, err := s.claim.GetNumComputeUnits()
	require.NoError(t, err)

	tokenomicsParams := s.keepers.Keeper.GetParams(ctx)
	// -1 to push threshold below s.claim's compute units
	proofRequirementThreshold, err := tokenomics.NumComputeUnitsToCoin(tokenomicsParams, numComputeUnits-1)
	require.NoError(t, err)

	// Set the proof parameters such that s.claim requires a proof because:
	// - proof_request_probability is 0%
	// - proof_requirement_threshold is below the claim (i.e. claim is above threshold)
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	// Set the proof missing penalty to be equal to the supplier's stake to make
	// its stake below the minimum stake requirement and trigger an unstake.
	proofParams.ProofMissingPenalty = &sdk.Coin{Denom: volatile.DenomuPOKT, Amount: math.NewInt(supplierStake)}
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Upsert the claim ONLY because it should be processed without needing a proof.
	s.keepers.UpsertClaim(ctx, s.claim)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should expire.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := s.claim.SessionHeader.SessionEndBlockHeight
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	_, _, err = s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	upcomingSessionEndHeight := uint64(shared.GetNextSessionStartHeight(&sharedParams, int64(blockHeight))) - 1

	// Slashing should have occurred and the supplier is unstaked but still unbonding.
	slashedSupplier, supplierFound := s.keepers.GetSupplier(sdkCtx, s.claim.SupplierOperatorAddress)
	require.True(t, supplierFound)
	require.Equal(t, math.NewInt(0), slashedSupplier.Stake.Amount)
	require.Equal(t, upcomingSessionEndHeight, slashedSupplier.UnstakeSessionEndHeight)
	require.True(t, slashedSupplier.IsUnbonding())

	events := sdkCtx.EventManager().Events()

	// Confirm that a slashing event was emitted
	expectedSlashingEvents := testutilevents.FilterEvents[*tokenomicstypes.EventSupplierSlashed](t, events, "poktroll.tokenomics.EventSupplierSlashed")
	require.Len(t, expectedSlashingEvents, 1)

	// Validate the slashing event
	expectedSlashingEvent := expectedSlashingEvents[0]
	require.Equal(t, slashedSupplier.GetOperatorAddress(), expectedSlashingEvent.GetSupplierOperatorAddr())
	require.Equal(t, uint64(1), expectedSlashingEvent.GetNumExpiredClaims())
	require.Equal(t, proofParams.ProofMissingPenalty, expectedSlashingEvent.GetSlashingAmount())
}

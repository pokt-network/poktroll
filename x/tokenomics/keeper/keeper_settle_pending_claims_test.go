package keeper_test

import (
	"context"
	"math/big"
	"testing"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	testutilproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
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

const testServiceId = "svc1"

var supplierStakeAmt = 2 * suppliertypes.DefaultMinStake.Amount.Int64()

func init() {
	cmd.InitSDKConfig()
}

// TODO_TECHDEBT(@bryanchriswhite): Consolidate the setup for all tests that use TokenomicsModuleKeepers
type TestSuite struct {
	suite.Suite

	ctx     context.Context
	keepers keepertest.TokenomicsModuleKeepers
	claim   prooftypes.Claim
	proof   prooftypes.Proof

	numRelays                uint64
	numClaimedComputeUnits   uint64
	numEstimatedComputeUnits uint64
	claimedUpokt             cosmostypes.Coin
	relayMiningDifficulty    servicetypes.RelayMiningDifficulty
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

	supplierStake := types.NewCoin("upokt", math.NewInt(supplierStakeAmt))
	supplierServiceConfigs := []*sharedtypes.SupplierServiceConfig{{
		ServiceId: testServiceId,
		RevShare: []*sharedtypes.ServiceRevenueShare{{
			Address:            supplierOwnerAddr,
			RevSharePercentage: 100,
		}},
	}}
	supplier := sharedtypes.Supplier{
		OwnerAddress:    supplierOwnerAddr,
		OperatorAddress: supplierOwnerAddr,
		Stake:           &supplierStake,
		Services:        supplierServiceConfigs,
		ServiceConfigHistory: []*sharedtypes.ServiceConfigUpdate{{
			Services:             supplierServiceConfigs,
			EffectiveBlockHeight: 1,
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

	// Calculate the number of claimed compute units.
	s.numClaimedComputeUnits = s.numRelays * service.ComputeUnitsPerRelay

	targetNumRelays := s.keepers.ServiceKeeper.GetParams(sdkCtx).TargetNumRelays
	s.relayMiningDifficulty = servicekeeper.NewDefaultRelayMiningDifficulty(
		sdkCtx,
		s.keepers.Logger(),
		testServiceId,
		targetNumRelays,
		targetNumRelays,
	)

	// Calculate the number of estimated compute units.
	s.numEstimatedComputeUnits = getEstimatedComputeUnits(s.numClaimedComputeUnits, s.relayMiningDifficulty)

	// Calculate the claimed amount in uPOKT.
	sharedParams := s.keepers.SharedKeeper.GetParams(sdkCtx)
	s.claimedUpokt = getClaimedUpokt(sharedParams, s.numEstimatedComputeUnits, s.relayMiningDifficulty)

	blockHeaderHash := make([]byte, 0)
	expectedMerkleProofPath := protocol.GetPathForProof(blockHeaderHash, sessionHeader.SessionId)

	// Advance the block height to the earliest claim commit height.
	claimMsgHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
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

	// Upsert the claim only
	s.keepers.UpsertClaim(ctx, s.claim)

	// Settle pending claims while the session is still active.
	// Expectations: No claims should be settled because the session is still ongoing
	blockHeight := s.claim.SessionHeader.SessionEndBlockHeight - 2 // session is still active
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, expiredResults, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled or expired.
	require.Equal(t, uint64(0), settledResults.GetNumClaims())
	require.Equal(t, uint64(0), expiredResults.GetNumClaims())

	// Validate that one claim still remains.
	claims := s.keepers.GetAllClaims(ctx)
	// TODO_TECHDEBT(@bryanchriswhite): Ensure docusaurus docs include a note regarding
	// preferring `require.Equal()` over `require.Len()` due to poor developer experience
	// when debugging failing tests which use the latter. TL;DR, the error message prints
	// the list in one line and is difficult and slow to parse. Then reference the doc here.
	require.Equal(t, 1, len(claims))

	// Calculate a block height which is within the proof window.
	proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(
		&sharedParams, s.claim.SessionHeader.SessionEndBlockHeight,
	)
	proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(
		&sharedParams, s.claim.SessionHeader.SessionEndBlockHeight,
	)
	blockHeight = (proofWindowCloseHeight - proofWindowOpenHeight) / 2

	// 2. Settle pending claims just after the session ended.
	// Expectations: Claims should not be settled because the proof window hasn't closed yet.
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResults, expiredResults, err = s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled or expired.
	require.Equal(t, uint64(0), settledResults.GetNumClaims())
	require.Equal(t, uint64(0), expiredResults.GetNumClaims())

	// Validate that the claim still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Equal(t, 1, len(claims))
}

func (s *TestSuite) TestSettlePendingClaims_ClaimExpired_ProofRequiredAndNotProvided_ViaThreshold() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	proofRequirementThreshold, err := s.claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulty)
	require.NoError(t, err)

	// -1 to push threshold below s.claim's compute units
	proofRequirementThreshold = proofRequirementThreshold.Sub(uPOKTCoin(1))

	// Set the proof missing penalty to half the supplier's stake so it is not
	// unstaked when being slashed.
	belowStakeAmountProofMissingPenalty := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(supplierStakeAmt/2))

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
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, expiredResults, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(0), settledResults.GetNumClaims()) // 0 claims settled
	require.Equal(t, uint64(1), expiredResults.GetNumClaims()) // 1 claim expired

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Equal(t, 0, len(claims))

	// Slashing should have occurred without unstaking the supplier.
	// The supplier is not unstaked because it got slashed by an amount that is
	// half its stake (i.e. missing proof penalty == stake / 2), resulting in a
	// remaining stake that is above the minimum stake (i.e. new_stake == prev_stake / 2).
	slashedSupplier, supplierFound := s.keepers.GetSupplier(sdkCtx, s.claim.SupplierOperatorAddress)
	require.True(t, supplierFound)
	require.Equal(t, supplierStakeAmt/2, slashedSupplier.Stake.Amount.Int64())
	require.Equal(t, uint64(0), slashedSupplier.UnstakeSessionEndHeight)

	// Validate the supplier and tokenomics module balances.
	supplierModuleBalRes, err := s.keepers.Balance(s.ctx, &banktypes.QueryBalanceRequest{
		Address: authtypes.NewModuleAddress(suppliertypes.ModuleName).String(),
		Denom:   volatile.DenomuPOKT,
	})
	require.NoError(t, err)
	require.Equal(t, supplierStakeAmt/2, supplierModuleBalRes.Balance.Amount.Int64())

	tokenomicsModuleBalRes, err := s.keepers.Balance(s.ctx, &banktypes.QueryBalanceRequest{
		Address: authtypes.NewModuleAddress(tokenomicstypes.ModuleName).String(),
		Denom:   volatile.DenomuPOKT,
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
	require.Equal(t, s.claimedUpokt, *expectedClaimExpiredEvent.GetClaimedUpokt())

	// Confirm that a slashing event was emitted
	expectedSlashingEvents := testutilevents.FilterEvents[*tokenomicstypes.EventSupplierSlashed](t, events)
	require.Equal(t, 1, len(expectedSlashingEvents))

	// Validate the slashing event
	expectedSlashingEvent := expectedSlashingEvents[0]

	require.Equal(t, slashedSupplier.GetOperatorAddress(), expectedSlashingEvent.GetClaim().GetSupplierOperatorAddress())
	require.Equal(t, &belowStakeAmountProofMissingPenalty, expectedSlashingEvent.GetProofMissingPenalty())
}

func (s *TestSuite) TestSettlePendingClaims_ClaimSettled_ProofRequiredAndProvided_ViaThreshold() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	proofRequirementThreshold, err := s.claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulty)
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
	s.keepers.UpsertClaim(ctx, s.claim)
	s.keepers.UpsertProof(ctx, s.proof)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	s.keepers.ValidateSubmittedProofs(sdkCtx)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := s.claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(1), settledResult.GetNumClaims()) // 1 claim settled
	require.Equal(t, uint64(0), expiredResult.GetNumClaims()) // 0 claims expired

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
	require.Equal(t, s.claimedUpokt, *expectedEvent.GetClaimedUpokt())
}

func (s *TestSuite) TestSettlePendingClaims_ClaimExpired_ProofRequired_InvalidOneProvided() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	// Set the proof parameters such that s.claim DOES NOT require a proof because:
	// - proof_request_probability is 100%
	proofParams.ProofRequestProbability = 1
	// Set the proof missing penalty to half the supplier's stake so it is not
	// unstaked when being slashed.
	belowStakeAmountProofMissingPenalty := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(supplierStakeAmt/2))
	proofParams.ProofMissingPenalty = &belowStakeAmountProofMissingPenalty
	err := s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Create a claim that requires a proof and an invalid proof
	proof := s.proof
	proof.ClosestMerkleProof = []byte("invalid_proof")

	// Upsert the proof & claim
	s.keepers.UpsertClaim(ctx, s.claim)
	s.keepers.UpsertProof(ctx, proof)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	s.keepers.ValidateSubmittedProofs(sdkCtx)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be expired.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := s.claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResults, expiredResults, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(0), settledResults.GetNumClaims()) // 0 claims settled
	require.Equal(t, uint64(1), expiredResults.GetNumClaims()) // 1 claim expired

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Equal(t, 0, len(claims))

	// Validate that no proofs remain.
	proofs := s.keepers.GetAllProofs(ctx)
	require.Equal(t, 0, len(proofs))

	// Slashing should have occurred without unstaking the supplier.
	slashedSupplier, supplierFound := s.keepers.GetSupplier(sdkCtx, s.claim.SupplierOperatorAddress)
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
	require.Equal(t, s.claimedUpokt, *expectedClaimExpiredEvent.GetClaimedUpokt())

	expectedProofValidityCheckedEvent := expectedProofValidityCheckedEvents[0]
	require.Equal(t, prooftypes.ClaimProofStatus_INVALID, expectedProofValidityCheckedEvent.GetProofStatus())

	// Confirm that a slashing event was emitted
	expectedSlashingEvents := testutilevents.FilterEvents[*tokenomicstypes.EventSupplierSlashed](t, events)
	require.Equal(t, 1, len(expectedSlashingEvents))

	// Validate the slashing event
	expectedSlashingEvent := expectedSlashingEvents[0]
	require.Equal(t, slashedSupplier.GetOperatorAddress(), expectedSlashingEvent.GetClaim().GetSupplierOperatorAddress())
	require.Equal(t, &belowStakeAmountProofMissingPenalty, expectedSlashingEvent.GetProofMissingPenalty())
}

func (s *TestSuite) TestClaimSettlement_ClaimSettled_ProofRequiredAndProvided_ViaProbability() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	proofRequirementThreshold, err := s.claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulty)
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
	s.keepers.UpsertClaim(ctx, s.claim)
	s.keepers.UpsertProof(ctx, s.proof)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	s.keepers.ValidateSubmittedProofs(sdkCtx)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proof window has definitely closed at this point
	sessionEndHeight := s.claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, expiredResults, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Validate claim settlement results
	require.Equal(t, uint64(1), settledResults.GetNumClaims()) // 1 claim settled
	require.Equal(t, uint64(0), expiredResults.GetNumClaims()) // 0 claims expired

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
	require.Equal(t, s.claimedUpokt, *expectedEvent.GetClaimedUpokt())
}

func (s *TestSuite) TestSettlePendingClaims_Settles_WhenAProofIsNotRequired() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	proofRequirementThreshold, err := s.claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulty)
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
	s.keepers.UpsertClaim(ctx, s.claim)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	s.keepers.ValidateSubmittedProofs(sdkCtx)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := s.claim.SessionHeader.SessionEndBlockHeight
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResults, expiredResults, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that one claim was settled.
	require.Equal(t, uint64(1), settledResults.GetNumClaims()) // 1 claim settled
	require.Equal(t, uint64(0), expiredResults.GetNumClaims()) // 0 claims expired

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
	require.Equal(t, s.claimedUpokt, *expectedEvent.GetClaimedUpokt())
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

	proofRequirementThreshold, err := s.claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulty)
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

	sessionOneProofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionOneEndHeight)
	sessionTwoStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, sessionOneProofWindowCloseHeight+1)
	sessionTwoProofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionTwoStartHeight)

	sessionTwoClaim.SessionHeader = &sessiontypes.SessionHeader{
		ApplicationAddress:      sessionOneClaim.GetSessionHeader().GetApplicationAddress(),
		ServiceId:               s.claim.GetSessionHeader().GetServiceId(),
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
	settledResults, expiredResults, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that one claim was settled.
	require.Equal(t, uint64(1), settledResults.GetNumClaims())

	// Validate that no claims expired.
	require.Equal(t, uint64(0), expiredResults.GetNumClaims())

	// Validate that one claim still remains.
	claims = s.keepers.GetAllClaims(ctx)
	require.Equal(t, 1, len(claims))

	// Calculate a block height which is within session two's proof window.
	blockHeight = (sessionTwoProofWindowCloseHeight - sessionTwoStartHeight) / 2

	// 2. Settle pending claims just after the session ended.
	// Expectations: Claims should not be settled because the proof window hasn't closed yet.
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResults, expiredResults, err = s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled or expired.
	require.Equal(t, uint64(0), settledResults.GetNumClaims())
	require.Equal(t, uint64(0), expiredResults.GetNumClaims())

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

	proofRequirementThreshold, err := s.claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulty)
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
	proofParams.ProofMissingPenalty = &cosmostypes.Coin{Denom: volatile.DenomuPOKT, Amount: math.NewInt(supplierStakeAmt)}
	err = s.keepers.ProofKeeper.SetParams(ctx, proofParams)
	require.NoError(t, err)

	// Creating multiple claims without proofs to test the claim expiration process:
	// - We're creating exactly numExpiredClaims applications to create numExpiredClaims claims
	// - All claims are for the same settlement period
	// - This allows us to verify that only a single unbonding event is emitted
	//   despite having multiple expired claims for the same supplier
	expiredClaimsMap := make(map[string]*prooftypes.Claim, numExpiredClaims)
	for range numExpiredClaims {
		appStake := types.NewCoin("upokt", math.NewInt(1000000))
		appAddr := sample.AccAddress()
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
		sessionRes, sessionErr := s.keepers.GetSession(sdkCtx, sessionReq)
		require.NoError(t, sessionErr)

		sessionHeader := sessionRes.Session.Header
		merkleRoot := testproof.SmstRootWithSumAndCount(1000, 1000)
		claim := testtree.NewClaim(t, s.claim.SupplierOperatorAddress, sessionHeader, merkleRoot)
		expiredClaimsMap[sessionHeader.SessionId] = claim
		s.keepers.UpsertClaim(ctx, *claim)
	}

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should expire.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := s.claim.SessionHeader.SessionEndBlockHeight
	sessionProofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = sdkCtx.WithBlockHeight(sessionProofWindowCloseHeight)
	_, _, err = s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	upcomingSessionEndHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, sessionProofWindowCloseHeight) - 1

	// Slashing should have occurred and the supplier is unstaked but still unbonding.
	slashedSupplier, supplierFound := s.keepers.GetSupplier(sdkCtx, s.claim.SupplierOperatorAddress)
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
		proofMissingPenalty := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)
		// The current slashing mechanism burns the whole supplier's stake, which
		// means that the first ocurrence will take the whole stake.
		if i == 0 {
			proofMissingPenalty = *s.keepers.ProofKeeper.GetParams(sdkCtx).ProofMissingPenalty
		}

		sessionId := slashingEvent.GetClaim().GetSessionHeader().GetSessionId()
		expectedSlashingEvent := &tokenomicstypes.EventSupplierSlashed{
			Claim:               expiredClaimsMap[sessionId],
			ProofMissingPenalty: &proofMissingPenalty,
		}
		require.EqualValues(t, expectedSlashingEvent, slashingEvents[i])
	}

	// Assert that an EventSupplierUnbondingBegin event was emitted.
	unbondingBeginEvents := testutilevents.FilterEvents[*suppliertypes.EventSupplierUnbondingBegin](t, events)
	require.Equal(t, 1, len(unbondingBeginEvents))

	// Validate the EventSupplierUnbondingBegin event.
	for i := range slashedSupplier.GetServices() {
		slashedSupplier.Services[i].Endpoints = make([]*sharedtypes.SupplierEndpoint, 0)
		slashedSupplier.ServiceConfigHistory[0].Services[i].Endpoints = make([]*sharedtypes.SupplierEndpoint, 0)
	}
	emptyServiceConfig := make([]*sharedtypes.SupplierServiceConfig, 0)
	slashedSupplier.ServiceConfigHistory[1].Services = emptyServiceConfig
	expectedUnbondingBeginEvent := &suppliertypes.EventSupplierUnbondingBegin{
		Supplier:           &slashedSupplier,
		Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE,
		SessionEndHeight:   upcomingSessionEndHeight,
		UnbondingEndHeight: upcomingSessionEndHeight,
	}
	// A signle unbonding begin event corresponding to the slashed supplier should be
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

	computeUnitsToTokenMultiplierRat := new(big.Rat).SetUint64(sharedParams.GetComputeUnitsToTokensMultiplier())

	claimedUpoktRat := new(big.Rat).Mul(numEstimatedComputeUnitsRat, computeUnitsToTokenMultiplierRat)
	claimedUpoktInt := new(big.Int).Div(claimedUpoktRat.Num(), claimedUpoktRat.Denom())

	return cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewIntFromBigInt(claimedUpoktInt))
}

// uPOKTCoin returns a uPOKT coin with the given amount.
func uPOKTCoin(amount int64) cosmostypes.Coin {
	return cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(amount))
}

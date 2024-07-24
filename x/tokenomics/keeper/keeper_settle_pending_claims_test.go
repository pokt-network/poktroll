package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	testutilproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testtree"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const testServiceId = "svc1"

func init() {
	cmd.InitSDKConfig()
}

type TestSuite struct {
	suite.Suite

	sdkCtx  cosmostypes.Context
	ctx     context.Context
	keepers keepertest.TokenomicsModuleKeepers
	claim   prooftypes.Claim
	proof   prooftypes.Proof

	expectedComputeUnits uint64
}

// SetupTest creates the following and stores them in the suite:
// - An cosmostypes.Context.
// - A keepertest.TokenomicsModuleKeepers to provide access to integrated keepers.
// - An expectedComputeUnits which is the default proof_requirement_threshold.
// - A claim that will require a proof via threshold, given the default proof params.
// - A proof which contains only the session header supplier address.
func (s *TestSuite) SetupTest() {
	t := s.T()

	s.keepers, s.ctx = keepertest.NewTokenomicsModuleKeepers(s.T(), nil)
	s.sdkCtx = cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(1)
	s.ctx = s.sdkCtx

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(s.keepers.Codec)

	// Create a pre-generated account iterator to create accounts for the test.
	preGeneratedAccts := testkeyring.PreGeneratedAccounts()

	// Create accounts in the account keeper with corresponding keys in the keyring
	// // for the applications and suppliers used in the tests.
	supplierAddr := testkeyring.CreateOnChainAccount(
		s.ctx, t,
		"supplier",
		keyRing,
		s.keepers.AccountKeeper,
		preGeneratedAccts,
	).String()
	appAddr := testkeyring.CreateOnChainAccount(
		s.ctx, t,
		"app",
		keyRing,
		s.keepers.AccountKeeper,
		preGeneratedAccts,
	).String()

	service := &sharedtypes.Service{Id: testServiceId}

	supplierStake := types.NewCoin("upokt", math.NewInt(1000000))
	supplier := sharedtypes.Supplier{
		Address:  supplierAddr,
		Stake:    &supplierStake,
		Services: []*sharedtypes.SupplierServiceConfig{{Service: service}},
	}
	s.keepers.SetSupplier(s.ctx, supplier)

	appStake := types.NewCoin("upokt", math.NewInt(1000000))
	app := apptypes.Application{
		Address:        appAddr,
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{Service: service}},
	}
	s.keepers.SetApplication(s.ctx, app)

	// Get the session for the application/supplier pair which is expected
	// to be claimed and for which a valid proof would be accepted.
	sessionReq := &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddr,
		Service:            service,
		BlockHeight:        1,
	}
	sessionRes, err := s.keepers.GetSession(s.sdkCtx, sessionReq)
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

	// Construct a valid session tree with 10 relays.
	numRelays := uint(10)
	sessionTree := testtree.NewFilledSessionTree(
		s.ctx, t,
		numRelays,
		"supplier", supplierAddr,
		sessionHeader, sessionHeader, sessionHeader,
		keyRing,
		ringClient,
	)
	s.expectedComputeUnits = testtree.FillSessionTreeExpectedComputeUnits(numRelays)

	blockHeaderHash := make([]byte, 0)
	expectedMerkleProofPath := protocol.GetPathForProof(blockHeaderHash, sessionHeader.SessionId)

	// Advance the block height to the earliest claim commit height.
	sharedParams := s.keepers.SharedKeeper.GetParams(s.ctx)
	claimMsgHeight := shared.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionHeader.GetSessionEndBlockHeight(),
		blockHeaderHash,
		supplierAddr,
	)
	s.sdkCtx = cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(claimMsgHeight).WithHeaderHash(blockHeaderHash
	s.ctx = s.sdkCtx


	merkleRootBz, err := sessionTree.Flush()
	require.NoError(t, err)

	// Prepare a claim that can be inserted
	s.claim = *testtree.NewClaim(t, supplierAddr, sessionHeader, merkleRootBz)
	s.proof = *testtree.NewProof(t, supplierAddr, sessionHeader, sessionTree, expectedMerkleProofPath)
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
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// 0. Add the claim & verify it exists
	claim := s.claim
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// 1. Settle pending claims while the session is still active.
	// Expectations: No claims should be settled because the session is still ongoing
	blockHeight := claim.SessionHeader.SessionEndBlockHeight - 2 // session is still active
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled or expired.
	require.Equal(t, uint64(0), settledResult.NumClaims)
	require.Equal(t, uint64(0), expiredResult.NumClaims)

	// Validate that one claim still remains.
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)

	// Calculate a block height which is within the proof window.
	proofWindowOpenHeight := shared.GetProofWindowOpenHeight(
		&sharedParams, claim.SessionHeader.SessionEndBlockHeight,
	)
	proofWindowCloseHeight := shared.GetProofWindowCloseHeight(
		&sharedParams, claim.SessionHeader.SessionEndBlockHeight,
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
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Create a claim that requires a proof
	claim := s.claim

	// 0. Add the claim & verify it exists
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be expired.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled.
	require.Equal(t, uint64(0), settledResult.NumClaims)
	// Validate that exactly one claims expired
	require.Equal(t, uint64(1), expiredResult.NumClaims)

	// Validate that no claims remain.
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an expiration event was emitted
	events := sdkCtx.EventManager().Events()
	require.Len(t, events, 5) // minting, burning, settling, etc..
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimExpired](t,
		events, "poktroll.tokenomics.EventClaimExpired")
	require.Len(t, expectedEvents, 1)

	// Validate the event
	expectedEvent := expectedEvents[0]
	require.Equal(t, tokenomicstypes.ClaimExpirationReason_PROOF_MISSING, expectedEvent.GetExpirationReason())
	require.Equal(t, s.expectedComputeUnits, expectedEvent.GetNumComputeUnits())
}

func (s *TestSuite) TestSettlePendingClaims_ClaimExpired_ProofRequired_InvalidOneProvided() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Create a claim that requires a proof and an invalid proof
	claim := s.claim
	proof := s.proof
	proof.ClosestMerkleProof = []byte("invalid_proof")

	// Upsert the proof & claim
	s.keepers.UpsertClaim(ctx, claim)
	s.keepers.UpsertProof(ctx, proof)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be expired.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled.
	require.Equal(t, uint64(0), settledResult.NumClaims)
	// Validate that exactly one claims expired
	require.Equal(t, uint64(1), expiredResult.NumClaims)

	// Validate that no claims remain.
	claims := s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Validate that no proofs remain.
	proofs := s.keepers.GetAllProofs(ctx)
	require.Len(t, proofs, 0)

	// Confirm an expiration event was emitted
	events := sdkCtx.EventManager().Events()
	require.Len(t, events, 5) // minting, burning, settling, etc..
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimExpired](t,
		events, "poktroll.tokenomics.EventClaimExpired")
	require.Len(t, expectedEvents, 1)

	// Validate the event
	expectedEvent := expectedEvents[0]
	require.Equal(t, tokenomicstypes.ClaimExpirationReason_PROOF_INVALID, expectedEvent.GetExpirationReason())
	require.Equal(t, s.expectedComputeUnits, expectedEvent.GetNumComputeUnits())
}

func (s *TestSuite) TestSettlePendingClaims_ClaimSettled_ProofRequiredAndProvided_ViaThreshold() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Create a claim that requires a proof
	claim := s.claim

	// Add the claim & verify it exists
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// Upsert the proof
	s.keepers.UpsertProof(ctx, s.proof)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that one claim was settled.
	require.Equal(t, uint64(1), settledResult.NumClaims)

	// Validate that no claims expired.
	require.Equal(t, uint64(0), expiredResult.NumClaims)

	// Validate that no claims remain.
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an settlement event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t,
		events, "poktroll.tokenomics.EventClaimSettled")
	require.Len(t, expectedEvents, 1)

	// Validate the event
	expectedEvent := expectedEvents[0]
	require.Equal(t, prooftypes.ProofRequirementReason_THRESHOLD, expectedEvent.GetProofRequirement())
	require.Equal(t, s.expectedComputeUnits, expectedEvent.GetNumComputeUnits())
}

func (s *TestSuite) TestClaimSettlement_ClaimSettled_ProofRequiredAndProvided_ViaProbability() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Set the proof parameters such that s.claim requires a proof because the
	// proof_request_probability is 100%. This is accomplished by setting the
	// proof_requirement_threshold to exceed s.expectedComputeUnits, which
	// matches s.claim.
	err := s.keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability: 1,
		// +1 to push the requirement threshold ABOVE s.claim's compute units
		ProofRequirementThreshold: s.expectedComputeUnits + 1,
	})
	require.NoError(t, err)

	// Create a claim that requires a proof
	claim := s.claim

	// 0. Add the claim & verify it exists
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// Upsert the proof
	s.keepers.UpsertProof(ctx, s.proof)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proof window has definitely closed at this point
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that one claim was settled.
	require.Equal(t, uint64(1), settledResult.NumClaims)
	// Validate that no claims expired.
	require.Equal(t, uint64(0), expiredResult.NumClaims)

	// Validate that no claims remain.
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an settlement event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t,
		events, "poktroll.tokenomics.EventClaimSettled")
	require.Len(t, expectedEvents, 1)
	expectedEvent := expectedEvents[0]
	require.Equal(t, prooftypes.ProofRequirementReason_PROBABILISTIC, expectedEvent.GetProofRequirement())
	require.Equal(t, s.expectedComputeUnits, expectedEvent.GetNumComputeUnits())
}

func (s *TestSuite) TestSettlePendingClaims_Settles_WhenAProofIsNotRequired() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Create a claim that does not require a proof
	claim := s.claim

	// Set the proof parameters such that s.claim DOES NOT require a proof because
	// the proof_request_probability is 0% AND because the proof_requirement_threshold
	// exceeds s.expectedComputeUnits, which matches s.claim.
	err := s.keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability: 0,
		// +1 to push the threshold above s.claim's compute units
		ProofRequirementThreshold: s.expectedComputeUnits + 1,
	})
	require.NoError(t, err)

	// Add the claim & verify it exists
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	sessionEndHeight := claim.SessionHeader.SessionEndBlockHeight
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	settledResult, expiredResult, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that one claim was settled.
	require.Equal(t, uint64(1), settledResult.NumClaims)
	// Validate that no claims expired.
	require.Equal(t, uint64(0), expiredResult.NumClaims)

	// Validate that no claims remain.
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an expiration event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t,
		events, "poktroll.tokenomics.EventClaimSettled")
	require.Len(t, expectedEvents, 1)

	// Validate the event
	expectedEvent := expectedEvents[0]
	require.Equal(t, prooftypes.ProofRequirementReason_NOT_REQUIRED.String(), expectedEvent.GetProofRequirement().String())
	require.Equal(t, s.expectedComputeUnits, expectedEvent.GetNumComputeUnits())
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

	// Set the proof parameters such that s.claim DOES NOT require a proof
	// because the proof_request_probability is 0% and the proof_request_threshold
	// is greater than the claims' compute units.
	err := s.keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability: 0,
		// +1 to push the threshold above s.claim's compute units
		ProofRequirementThreshold: s.expectedComputeUnits + 1,
	})
	require.NoError(t, err)

	// 0. Add the claims & verify they exists
	sessionOneClaim := s.claim
	s.keepers.UpsertClaim(ctx, sessionOneClaim)

	sessionOneEndHeight := sessionOneClaim.GetSessionHeader().GetSessionEndBlockHeight()

	// Add a second claim with a session header corresponding to the next session.
	sessionTwoClaim := testutilproof.BaseClaim(
		sessionOneClaim.GetSessionHeader().GetApplicationAddress(),
		sessionOneClaim.GetSupplierAddress(),
		s.expectedComputeUnits,
	)

	sessionOneProofWindowCloseHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionOneEndHeight)
	sessionTwoStartHeight := shared.GetSessionStartHeight(&sharedParams, sessionOneProofWindowCloseHeight+1)
	sessionTwoProofWindowCloseHeight := shared.GetProofWindowCloseHeight(&sharedParams, sessionTwoStartHeight)

	sessionTwoClaim.SessionHeader = &sessiontypes.SessionHeader{
		ApplicationAddress:      sessionOneClaim.GetSessionHeader().GetApplicationAddress(),
		Service:                 s.claim.GetSessionHeader().GetService(),
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

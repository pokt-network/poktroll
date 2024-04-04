package keeper_test

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	testServiceId = "svc1"
	testSessionId = "mock_session_id"
)
const minExecutionPeriod = 5 * time.Second

func init() {
	cmd.InitSDKConfig()
}

type TestSuite struct {
	suite.Suite

	sdkCtx  sdk.Context
	ctx     context.Context
	keepers keepertest.TokenomicsModuleKeepers
	claim   prooftypes.Claim
}

func (s *TestSuite) SetupTest() {
	supplierAddr := sample.AccAddress()
	appAddr := sample.AccAddress()

	// Prepare and insert the claim
	s.claim = prooftypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      appAddr,
			Service:                 &sharedtypes.Service{Id: testServiceId},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   sessionkeeper.GetSessionEndBlockHeight(1),
		},
		RootHash: []byte("default_roo_hash"),
	}

	s.keepers, s.ctx = keepertest.NewTokenomicsModuleKeepers(s.T())
	s.sdkCtx = sdk.UnwrapSDKContext(s.ctx)
}

func TestSettleExpiringSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestClaimSettlesWhenAProofIsRequired() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	claim := s.claim

	// Add the claim & verify it exists
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// Settle expiring claims at height 2 (while the session is still active).
	// Expectations: No claims should be settled because the session is still ongoing
	sdkCtx = sdkCtx.WithBlockHeight(claim.SessionHeader.SessionEndBlockHeight - 2)
	numClaimsSettled, numClaimsExpired, err := s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(0), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Validate that the claim still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)

	// Try to settle expiring claim a little after it ended.
	// Goal: Claims should not be settled because the proof window hasn't closed yet.
	sdkCtx = sdkCtx.WithBlockHeight(claim.SessionHeader.SessionEndBlockHeight + 2)
	numClaimsSettled, numClaimsExpired, err = s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(0), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Validate that the claim still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)

	// Try to settle expiring claims a long time after it ended
	// Expectation: All (1) claims should be settled.
	sdkCtx = sdkCtx.WithBlockHeight(claim.SessionHeader.SessionEndBlockHeight * 10)
	numClaimsSettled, numClaimsExpired, err = s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(0), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Validate that the claims expired
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an expiration event was emitted
	events := sdkCtx.EventManager().Events()
	require.Len(t, events, 1)
}

func (s *TestSuite) TestClaimSettlesWhenAProofIsNotRequired() {
	s.T().Skip("TODO_TEST: Implement that a claim is properly settled when a claim is provided but a proof is not needed for it")
}

func (s *TestSuite) TestClaimDoesNotSettleBeforeProofWindowCloses() {
	s.T().Skip("TODO_TEST: Implement that a claim remains unsettled before the proof window closes")
}

func (s *TestSuite) TestClaimDoesNotSettleIfProofIsInvalid() {
	s.T().Skip("TODO_TEST: Implement that a claim remains unsettled before the proof window closes")
}

func (s *TestSuite) TestClaimDoesNotSettleIfProofIsRequiredButMissing() {
	s.T().Skip("TODO_TEST: Implement that a claim remains unsettled before the proof window closes")
}

func (s *TestSuite) TestMultipleClaimsSettleWithMultipleApplicationsAndSuppliers() {
	s.T().Skip("TODO_TEST: Implement that multiple claims settle at once when different sessions have overlapping applications and suppliers")
}

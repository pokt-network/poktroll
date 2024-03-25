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
}

func (s *TestSuite) SetupTest() {
	s.keepers, s.ctx = keepertest.NewTokenomicsModuleKeepers(s.T())
	s.sdkCtx = sdk.UnwrapSDKContext(s.ctx)
}

func TestSettleExpiringSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestClaimWithoutProofExpires() {
	t := s.T()
	supplierAddr := sample.AccAddress()
	appAddr := sample.AccAddress()

	ctx := s.ctx
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Prepare and insert the claim
	claim := prooftypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      appAddr,
			Service:                 &sharedtypes.Service{Id: testServiceId},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   1 + sessionkeeper.NumBlocksPerSession,
		},
		RootHash: []byte("root_hash"),
	}
	s.keepers.UpsertClaim(ctx, claim)

	// Verify that the claim exists
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// Settle expiring claims at height 2 (while the session is still active).
	// Expectations: No claims should be settled.
	sdkCtx = sdkCtx.WithBlockHeight(2)
	numClaimsSettled, numClaimsExpired, err := s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(0), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Check that the claims still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)

	// Try to settle expiring claims at height 2 (while the session is still active).
	// Goal: Claims should not be settled.
	sdkCtx = sdkCtx.WithBlockHeight(5)
	numClaimsSettled, numClaimsExpired, err = s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(0), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Check that the claims still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)

	// Try to settle expiring claims at height 20 (after the proof window closes).
	// Expectation: All (1) claims should be settled.
	sdkCtx = sdkCtx.WithBlockHeight(20)
	numClaimsSettled, numClaimsExpired, err = s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(0), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Check that the claims expired
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	events := sdkCtx.EventManager().Events()
	require.Len(t, events, 1)
}

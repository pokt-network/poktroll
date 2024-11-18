package token_logic_modules

import (
	"context"
	"math"
	"testing"

	"cosmossdk.io/depinject"
	cosmoslog "cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testtree"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

type tokenLogicModuleTestSuite struct {
	suite.Suite

	ctx     context.Context
	keepers testkeeper.TokenomicsModuleKeepers

	service  *sharedtypes.Service
	app      *apptypes.Application
	supplier *sharedtypes.Supplier

	proposerConsAddr cosmostypes.ConsAddress
	sourceOwnerAddr,
	daoRewardAddr string

	daoRewardAcct,
	proposerAcct,
	supplierAcct,
	appAcct *testkeyring.PreGeneratedAccount

	merkleProofPathSeed []byte
	supplierOperatorUid string
	keyRing             keyring.Keyring
	preGeneratedAccts   *testkeyring.PreGeneratedAccountIterator
	ringClient          crypto.RingClient

	expectedSettledResults,
	expectedExpiredResults tlm.SettlementResults
	expectedSettlementState *settlementState
}

// settlementState holds the expected post-settlement app stake and rewardee balances.
type settlementState struct {
	appModuleBalance        *cosmostypes.Coin
	supplierModuleBalance   *cosmostypes.Coin
	tokenomicsModuleBalance *cosmostypes.Coin

	appStake             *cosmostypes.Coin
	supplierOwnerBalance *cosmostypes.Coin
	proposerBalance      *cosmostypes.Coin
	daoBalance           *cosmostypes.Coin
	sourceOwnerBalance   *cosmostypes.Coin
}

func init() {
	cmd.InitSDKConfig()
}

func TestTokenLogicModuleTestSuite(t *testing.T) {
	suite.Run(t, new(tokenLogicModuleTestSuite))
}

// SetupTest generates and sets all rewardee addresses on the suite, and
// set a service, application, and supplier on the suite.
func (s *tokenLogicModuleTestSuite) SetupTest() {
	s.supplierOperatorUid = "supplier-operator"
	s.merkleProofPathSeed = []byte("mock-merkle-proof-path-seed-block-hash")
}

// setupKeepers initializes a new instance of TokenomicsModuleKeepers and context
// with the given options, and creates the suite's service, application, and supplier
// from SetupTest(). It also sets the block height to 1 and the proposer address to
// the proposer address from SetupTest().
func (s *tokenLogicModuleTestSuite) setupKeepers(t *testing.T, opts ...testkeeper.TokenomicsModuleKeepersOptFn) {
	t.Helper()

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	s.keyRing = keyring.NewInMemory(cdc)
	s.preGeneratedAccts = testkeyring.PreGeneratedAccounts()

	s.sourceOwnerAddr = s.preGeneratedAccts.MustNext().Address.String()
	s.service = &sharedtypes.Service{
		Id:                   "svc1",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         s.sourceOwnerAddr,
	}

	s.setupActors(t)
	s.setupKeyRing(t)

	s.keepers, s.ctx = testkeeper.NewTokenomicsModuleKeepers(
		t, cosmoslog.NewTestLogger(t),
		append([]testkeeper.TokenomicsModuleKeepersOptFn{
			testkeeper.WithRegistry(registry),
			testkeeper.WithProposerAddr(s.proposerConsAddr.String()),
			testkeeper.WithService(*s.service),
			testkeeper.WithSupplier(*s.supplier),
			testkeeper.WithApplication(*s.app),
		}, opts...)...,
	)

	tokenomicsParams := tokenomicstypes.DefaultParams()
	tokenomicsParams.DaoRewardAddress = s.daoRewardAddr
	err := s.keepers.Keeper.SetParams(s.ctx, tokenomicsParams)
	require.NoError(t, err)

	// Increment the block height to 1; valid session height.
	s.setBlockHeight(1)

	// On-chain accounts MUST be created after keeper construction.
	s.setupOnChainAccounts(t)

	ringClientDeps := depinject.Supply(
		polyzero.NewLogger(polyzero.WithLevel(polyzero.DebugLevel)),
		prooftypes.NewAppKeeperQueryClient(s.keepers),
		prooftypes.NewAccountKeeperQueryClient(s.keepers),
		prooftypes.NewSharedKeeperQueryClient(s.keepers.SharedKeeper, s.keepers),
	)
	s.ringClient, err = rings.NewRingClient(ringClientDeps)
	require.NoError(s.T(), err)
}

// TODO_IH_THIS_COMMIT: move & godoc...
func (s *tokenLogicModuleTestSuite) setupActors(t *testing.T) {
	t.Helper()

	// TODO_IN_THIS_COMMIT: comment... must be 1st account...
	s.daoRewardAcct = s.preGeneratedAccts.MustNext()
	s.daoRewardAddr = s.daoRewardAcct.Address.String()

	s.proposerAcct = s.preGeneratedAccts.MustNext()
	s.proposerConsAddr = sample.ConsAddrFromAccBech32(s.proposerAcct.Address.String())

	s.supplierAcct = s.preGeneratedAccts.MustNext()
	supplierAddr := s.supplierAcct.Address.String()
	s.supplier = &sharedtypes.Supplier{
		OwnerAddress:    supplierAddr,
		OperatorAddress: supplierAddr,
		Stake:           &suppliertypes.DefaultMinStake,
		Services: []*sharedtypes.SupplierServiceConfig{
			{
				ServiceId: s.service.GetId(),
				RevShare: []*sharedtypes.ServiceRevenueShare{
					{
						Address:            supplierAddr,
						RevSharePercentage: 100,
					},
				},
			},
		},
		ServicesActivationHeightsMap: map[string]uint64{
			s.service.GetId(): 0,
		},
	}

	s.appAcct = s.preGeneratedAccts.MustNext()
	s.app = &apptypes.Application{
		Address: s.appAcct.Address.String(),
		Stake:   &apptypes.DefaultMinStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: s.service.GetId()},
		},
	}
}

// TODO_IH_THIS_COMMIT: move & godoc...
func (s *tokenLogicModuleTestSuite) setupKeyRing(t *testing.T) {
	t.Helper()

	// Add the dao reward account to the keyring.
	err := s.daoRewardAcct.AddToKeyring(s.keyRing, "dao-reward")
	require.NoError(s.T(), err)

	// Add the proposer account to the keyring.
	err = s.proposerAcct.AddToKeyring(s.keyRing, "proposer")
	require.NoError(s.T(), err)

	// Add the supplier account to the keyring.
	err = s.supplierAcct.AddToKeyring(s.keyRing, s.supplierOperatorUid)
	require.NoError(s.T(), err)

	// Add the application account to the keyring.
	err = s.appAcct.AddToKeyring(s.keyRing, "app")
	require.NoError(s.T(), err)
}

// TODO_IH_THIS_COMMIT: move & godoc...
func (s *tokenLogicModuleTestSuite) setupOnChainAccounts(t *testing.T) {
	t.Helper()

	// Create an on-chain dao reward account.
	err := s.daoRewardAcct.AddToAccountKeeper(s.ctx, s.keepers)
	require.NoError(t, err)

	// Create an on-chain proposer account.
	err = s.proposerAcct.AddToAccountKeeper(s.ctx, s.keepers)
	require.NoError(t, err)

	// Create an on-chain supplier account.
	err = s.supplierAcct.AddToAccountKeeper(s.ctx, s.keepers)
	require.NoError(t, err)

	// Create an on-chain application account.
	err = s.appAcct.AddToAccountKeeper(s.ctx, s.keepers)
	require.NoError(t, err)
}

// getProofParams returns the default proof params with a high proof requirement threshold
// and no proof request probability such that no claims require a proof.
func (s *tokenLogicModuleTestSuite) getProofParams() *prooftypes.Params {
	proofParams := prooftypes.DefaultParams()
	highProofRequirementThreshold := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, math.MaxInt64)
	proofParams.ProofRequirementThreshold = &highProofRequirementThreshold
	proofParams.ProofRequestProbability = 0
	return &proofParams
}

// getSharedParams returns the default shared params with the CUTTM set to 1.
func (s *tokenLogicModuleTestSuite) getSharedParams() *sharedtypes.Params {
	sharedParams := sharedtypes.DefaultParams()
	sharedParams.ComputeUnitsToTokensMultiplier = 1
	return &sharedParams
}

// getTokenomicsParams returns the default tokenomics params with the dao_reward_address set to s.daoRewardAddr.
func (s *tokenLogicModuleTestSuite) getTokenomicsParams() *tokenomicstypes.Params {
	tokenomicsParams := tokenomicstypes.DefaultParams()
	tokenomicsParams.DaoRewardAddress = s.daoRewardAddr
	return &tokenomicsParams
}

// createClaim creates numClaims number of claims for the current session given
// the suites service, application, and supplier.
// DEV_NOTE: The sum/count must be large enough to avoid a proposer reward
// (or other small proportion rewards) from being truncated to zero (> 1upokt).
func (s *tokenLogicModuleTestSuite) createClaims(numClaims int) {
	s.T().Helper()

	session, err := s.getSession()
	require.NoError(s.T(), err)

	// Create claims (no proof requirements)
	for i := 0; i < numClaims; i++ {
		claim := s.newClaim(session)
		s.keepers.UpsertClaim(s.ctx, *claim)
	}
}

// settleClaims sets the block height to the settlement height for the current
// session and triggers the settlement of all pending claims.
func (s *tokenLogicModuleTestSuite) settleClaims(t *testing.T) (settledResults, expiredResults tlm.SettlementResults) {
	settledPendingResults, expiredPendingResults, err := s.trySettleClaims()
	require.NoError(t, err)

	require.NotZero(t, len(settledPendingResults))
	// TODO_IMPROVE: enhance the test scenario to include expiring claims to increase coverage.
	require.Zero(t, len(expiredPendingResults))

	return settledPendingResults, expiredPendingResults
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (s *tokenLogicModuleTestSuite) trySettleClaims() (settledResults, expiredResults tlm.SettlementResults, err error) {
	// Increment the block height to the settlement height.
	settlementHeight := sharedtypes.GetSettlementSessionEndHeight(s.getSharedParams(), 1)
	s.setBlockHeight(settlementHeight)

	return s.keepers.SettlePendingClaims(cosmostypes.UnwrapSDKContext(s.ctx))
}

// setBlockHeight sets the block height of the suite's context to height.
func (s *tokenLogicModuleTestSuite) setBlockHeight(height int64) {
	s.ctx = cosmostypes.UnwrapSDKContext(s.ctx).WithBlockHeight(height)
}

// assertNoPendingClaims asserts that no pending claims exist.
func (s *tokenLogicModuleTestSuite) assertNoPendingClaims(t *testing.T) {
	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx)
	pendingClaims, err := s.keepers.GetExpiringClaims(sdkCtx)
	require.NoError(t, err)
	require.Zero(t, len(pendingClaims))
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *tokenLogicModuleTestSuite) getSession() (*sessiontypes.Session, error) {
	sessionRes, err := s.keepers.GetSession(s.ctx, &sessiontypes.QueryGetSessionRequest{
		ServiceId:          s.service.GetId(),
		ApplicationAddress: s.app.GetAddress(),
		BlockHeight:        1,
	})
	if err != nil {
		return nil, err
	}

	return sessionRes.GetSession(), nil
}

// TODO_IN_THIS_COMMIT: godoc... AND inline; is this used anywhere else?
func (s *tokenLogicModuleTestSuite) newClaim(session *sessiontypes.Session) *prooftypes.Claim {
	return &prooftypes.Claim{
		SupplierOperatorAddress: s.supplier.GetOperatorAddress(),
		SessionHeader:           session.GetHeader(),
		RootHash:                proof.SmstRootWithSumAndCount(1000, 1000),
	}
}

// TODO_IN_THIS_COMMIT: move & godoc...
func (s *tokenLogicModuleTestSuite) newProof(t *testing.T, claim *prooftypes.Claim) *prooftypes.Proof {
	sessionHeader := claim.GetSessionHeader()
	numRelays := uint64(5)
	sessionTree := testtree.NewFilledSessionTree(
		s.ctx, t,
		numRelays, s.service.ComputeUnitsPerRelay,
		s.supplierOperatorUid, claim.GetSupplierOperatorAddress(),
		sessionHeader, sessionHeader, sessionHeader,
		s.keyRing,
		s.ringClient,
	)

	merkleRootBz, err := sessionTree.Flush()
	require.NoError(t, err)

	// Override the claim root hash with a valid one which corresponds to the unmangled proof.
	claim.RootHash = merkleRootBz

	merkleProofPath := protocol.GetPathForProof(
		s.merkleProofPathSeed,
		sessionTree.GetSessionHeader().GetSessionId(),
	)

	// Construct a proof message with a session tree containing
	// a valid relay but of insufficient difficulty.
	return testtree.NewProof(t,
		claim.GetSupplierOperatorAddress(),
		sessionHeader,
		sessionTree,
		merkleProofPath,
	)
}

// settleClaims sets the block height to the settlement height for the current
// session and triggers the settlement of all pending claims.
func (s *tokenLogicModuleTestSuite) settleClaims(t *testing.T) (settledResults, expiredResults tlm.PendingSettlementResults) {
	settledPendingResults, expiredPendingResults, err := s.trySettleClaims()
	require.NoError(t, err)

	require.NotZero(t, len(settledPendingResults))
	// TODO_IMPROVE: enhance the test scenario to include expiring claims to increase coverage.
	require.Zero(t, len(expiredPendingResults))

	return settledPendingResults, expiredPendingResults
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *tokenLogicModuleTestSuite) trySettleClaims() (settledResults, expiredResults tlm.PendingSettlementResults, err error) {
	// Increment the block height to the settlement height.
	settlementHeight := sharedtypes.GetSettlementSessionEndHeight(s.getSharedParams(), 1)
	s.setBlockHeight(settlementHeight)

	return s.keepers.SettlePendingClaims(cosmostypes.UnwrapSDKContext(s.ctx))
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *tokenLogicModuleTestSuite) getBlockHeight() int64 {
	return cosmostypes.UnwrapSDKContext(s.ctx).BlockHeight()
}

// setBlockHeight sets the block height of the suite's context to height.
func (s *tokenLogicModuleTestSuite) setBlockHeight(height int64) {
	s.ctx = cosmostypes.UnwrapSDKContext(s.ctx).
		WithBlockHeight(height).
		WithHeaderHash(s.merkleProofPathSeed).
		WithProposer(s.proposerConsAddr)
	s.keepers.StoreBlockHash(s.ctx)
}

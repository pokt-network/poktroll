package session_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"cosmossdk.io/depinject"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/gogo/status"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/supplier"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testqueryclients"
	"github.com/pokt-network/poktroll/testutil/testpolylog"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SessionPersistenceTestSuite defines the test suite for session persistence
type SessionPersistenceTestSuite struct {
	suite.Suite
	ctx          context.Context
	tmpStoresDir string

	deps                   depinject.Config
	relayerSessionsManager relayer.RelayerSessionsManager
	storesDirectoryOpt     relayer.RelayerSessionsManagerOption

	sessionTrees            session.SessionsTreesMap
	activeSessionHeader     *sessiontypes.SessionHeader
	supplierOperatorAddress string
	service                 sharedtypes.Service
	emptyBlockHash          []byte
	claimToReturn           *prooftypes.Claim
	createClaimCallCount    int
	submitProofCallCount    int
	latestBlock             client.Block

	blockClient    client.BlockClient
	blockPublishCh chan<- client.Block
	blocksObs      observable.Observable[client.Block]

	minedRelaysPublishCh chan<- *relayer.MinedRelay
	minedRelaysObs       relayer.MinedRelaysObservable

	sharedParams sharedtypes.Params
	proofParams  prooftypes.Params

	logger polylog.Logger
}

// TestSessionPersistence executes the session persistence test suite
func TestSessionPersistence(t *testing.T) {
	suite.Run(t, new(SessionPersistenceTestSuite))
}

// SetupTest prepares the test environment before each test execution.
// It initializes all necessary components including loggers, services, session headers,
// and the relayer sessions manager needed for the test.
func (s *SessionPersistenceTestSuite) SetupTest() {
	// Initialize logger and context
	s.logger, s.ctx = testpolylog.NewLoggerWithCtx(context.Background(), polyzero.DebugLevel)

	// Initialize test data and state
	s.service = sharedtypes.Service{Id: "svc", ComputeUnitsPerRelay: 2}
	s.sharedParams = sharedtypes.DefaultParams()
	s.proofParams = prooftypes.DefaultParams()
	s.proofParams.ProofRequirementThreshold = uPOKTCoin(1)
	s.supplierOperatorAddress = sample.AccAddress()
	s.emptyBlockHash = make([]byte, 32)

	// Reset counters and state for each test
	s.createClaimCallCount = 0
	s.submitProofCallCount = 0
	s.claimToReturn = nil
	s.sessionTrees = make(session.SessionsTreesMap)
	s.latestBlock = nil

	// Set up temporary directory for session storage
	tmpDirPattern := fmt.Sprintf("%s_smt_kvstore", strings.ReplaceAll(s.T().Name(), "/", "_"))
	tmpStoresDir, err := os.MkdirTemp("", tmpDirPattern)
	require.NoError(s.T(), err)
	s.storesDirectoryOpt = session.WithStoresDirectory(tmpStoresDir)

	// Configure test service and difficulty
	testqueryclients.AddToExistingServices(s.T(), s.service)
	testqueryclients.SetServiceRelayDifficultyTargetHash(s.T(), s.service.Id, protocol.BaseRelayDifficultyHashBz)

	// Create a session header for testing
	s.activeSessionHeader = &sessiontypes.SessionHeader{
		SessionStartBlockHeight: 1,
		SessionEndBlockHeight:   int64(s.sharedParams.NumBlocksPerSession),
		ServiceId:               s.service.Id,
		SessionId:               "sessionId",
	}

	// Set up dependencies for the relayer sessions manager
	s.deps = s.setupSessionManagerDependencies()

	// Create mined relays observable and channel for publishing
	mrObs, minedRelaysPublishCh := channel.NewObservable[*relayer.MinedRelay]()
	s.minedRelaysObs = relayer.MinedRelaysObservable(mrObs)
	s.minedRelaysPublishCh = minedRelaysPublishCh

	// Initialize and start the relayer sessions manager
	s.relayerSessionsManager = s.setupNewRelayerSessionsManager()
	s.advanceToBlock(1)
	err = s.relayerSessionsManager.Start(s.ctx)
	require.NoError(s.T(), err)

	// Publish a test mined relay and wait for processing
	s.minedRelaysPublishCh <- testrelayer.NewUnsignedMinedRelay(s.T(), s.activeSessionHeader, s.supplierOperatorAddress)
	waitSimulateIO()

	// Verify the session tree is correctly initialized
	sessionTree := s.getActiveSessionTree()
	require.Equal(s.T(), sessionTree.GetSessionHeader(), s.activeSessionHeader)

	// Verify the session tree has one element (the mined relay)
	smstRoot := sessionTree.GetSMSTRoot()
	count, err := smstRoot.Count()
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint64(1), count)
}

// TearDownTest cleans up resources after each test execution
func (s *SessionPersistenceTestSuite) TearDownTest() {
	// Stop the relayer sessions manager
	s.relayerSessionsManager.Stop()
	// Delete all temporary files and directories created by the test on completion.
	_ = os.RemoveAll(s.tmpStoresDir)
}

// TestSaveAndRetrieveSession tests the persistence of session data across relayer restarts.
// It verifies that session state is correctly saved to storage and can be retrieved after
// the relayer sessions manager is stopped and restarted.
func (s *SessionPersistenceTestSuite) TestSaveAndRetrieveSession() {
	// Stop the current relayer sessions manager
	s.relayerSessionsManager.Stop()
	// Create a new relayer sessions manager.
	// Note: This does not load state from the store.
	s.relayerSessionsManager = s.setupNewRelayerSessionsManager()

	// Advance to block 2 while the relayer sessions manager is stopped
	s.advanceToBlock(2)

	// Start the new relayer sessions manager and load state from the store
	err := s.relayerSessionsManager.Start(s.ctx)
	require.NoError(s.T(), err)
	waitSimulateIO()

	// Verify the session tree was correctly loaded and matches the original session header
	sessionTree := s.getActiveSessionTree()
	require.Equal(s.T(), s.activeSessionHeader, sessionTree.GetSessionHeader())

	// Verify the session tree contains exactly one relay (preserved from earlier test setup)
	smstRoot := sessionTree.GetSMSTRoot()
	count, err := smstRoot.Count()
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint64(1), count)

	// Add a new mined relay to the session
	s.minedRelaysPublishCh <- testrelayer.NewUnsignedMinedRelay(s.T(), s.activeSessionHeader, s.supplierOperatorAddress)
	// Advance to block 3
	s.advanceToBlock(3)

	// Verify that the new relay is added, bringing the count to 2
	smstRoot = sessionTree.GetSMSTRoot()
	count, err = smstRoot.Count()
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint64(2), count)
}

// TestRestartAfterClaimWindowOpen tests session persistence when the relayer is restarted
// after the claim window opens but before a claim is created.
// This verifies that the relayer automatically creates a claim when restarted
// during the claim window period.
func (s *SessionPersistenceTestSuite) TestRestartAfterClaimWindowOpen() {
	// Get the session end height from the active session header
	sessionEndHeight := s.activeSessionHeader.GetSessionEndBlockHeight()

	// Calculate when the claim window opens for this session
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(&s.sharedParams, sessionEndHeight)
	// Move to one block before the claim window opens
	s.advanceToBlock(claimWindowOpenHeight - 1)

	// Verify the session tree exists and no claims have been created yet
	sessionTree := s.getActiveSessionTree()
	require.Equal(s.T(), s.activeSessionHeader, sessionTree.GetSessionHeader())
	require.Equal(s.T(), 0, s.createClaimCallCount)

	// Stop and recreate the relayer sessions manager
	s.relayerSessionsManager.Stop()
	s.relayerSessionsManager = s.setupNewRelayerSessionsManager()

	// Advance to the block where the claim window opens while the relayer sessions manager is stopped
	s.blockPublishCh <- testblock.NewAnyTimesBlock(s.T(), s.emptyBlockHash, claimWindowOpenHeight)
	s.advanceToBlock(claimWindowOpenHeight)

	// Start the new relayer sessions manager
	err := s.relayerSessionsManager.Start(s.ctx)
	require.NoError(s.T(), err)
	waitSimulateIO()

	// Verify the session tree is still correctly loaded
	sessionTree = s.getActiveSessionTree()
	require.Equal(s.T(), s.activeSessionHeader, sessionTree.GetSessionHeader())

	// Verify a claim root has been created after restart
	claimRoot := sessionTree.GetClaimRoot()
	require.NotNil(s.T(), claimRoot)

	// Verify the claim tree has exactly one relay recorded
	count, err := smt.MerkleSumRoot(claimRoot).Count()
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint64(1), count)

	// Verify createClaim was called exactly once
	require.Equal(s.T(), 1, s.createClaimCallCount)
}

// TestRestartAfterClaimSubmitted tests session persistence when the relayer is restarted
// after a claim has already been submitted.
// This verifies that the relayer doesn't duplicate claims and can proceed
// to the proof submission phase correctly.
func (s *SessionPersistenceTestSuite) TestRestartAfterClaimSubmitted() {
	// Get the session end height from the active session header
	sessionEndHeight := s.activeSessionHeader.GetSessionEndBlockHeight()

	// Calculate when the claim window opens for this session
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(&s.sharedParams, sessionEndHeight)
	// Move to the block where the claim window opens (which should trigger claim creation)
	s.advanceToBlock(claimWindowOpenHeight)

	// Verify the session tree exists and a claim has been created
	sessionTree := s.getActiveSessionTree()
	claimRoot := sessionTree.GetClaimRoot()
	require.NotNil(s.T(), claimRoot)
	require.Equal(s.T(), 1, s.createClaimCallCount)

	// Verify the claim tree has exactly one claim
	count, err := smt.MerkleSumRoot(claimRoot).Count()
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint64(1), count)

	// Stop and recreate the relayer sessions manager
	s.relayerSessionsManager.Stop()
	s.relayerSessionsManager = s.setupNewRelayerSessionsManager()

	// Advance to the next block after claim window open while the relayer sessions manager is stopped
	s.advanceToBlock(claimWindowOpenHeight + 1)

	// Start the new relayer sessions manager
	err = s.relayerSessionsManager.Start(s.ctx)
	require.NoError(s.T(), err)
	waitSimulateIO()

	// Verify the session tree is still correctly loaded
	sessionTree = s.getActiveSessionTree()
	require.Equal(s.T(), s.activeSessionHeader, sessionTree.GetSessionHeader())

	// Verify no proof has been submitted yet
	require.Equal(s.T(), 0, s.submitProofCallCount)

	// Calculate when the proof window closes for this session
	proofWindowOpenHeight := sharedtypes.GetProofWindowCloseHeight(&s.sharedParams, sessionEndHeight)
	// Move to one block before the proof window closes (which should trigger proof submission)
	s.advanceToBlock(proofWindowOpenHeight)

	// Verify the session tree has been removed and a proof was submitted
	require.Len(s.T(), s.sessionTrees, 0)
	require.Equal(s.T(), 1, s.submitProofCallCount)
}

// TestRestartAfterClaimWindowClose tests session persistence when the relayer is restarted
// after the claim window has closed but before any claims were created.
// This verifies that the relayer correctly handles sessions where the claim window is missed.
func (s *SessionPersistenceTestSuite) TestRestartAfterClaimWindowClose() {
	// Get the session end height from the active session header
	sessionEndHeight := s.activeSessionHeader.GetSessionEndBlockHeight()

	// Calculate when the claim window opens for this session
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(&s.sharedParams, sessionEndHeight)
	// Move to one block before the claim window opens
	s.advanceToBlock(claimWindowOpenHeight - 1)

	// Verify the session tree exists and no claims have been created
	require.Len(s.T(), s.sessionTrees, 1)
	require.Equal(s.T(), 0, s.createClaimCallCount)

	// Stop and recreate the relayer sessions manager
	s.relayerSessionsManager.Stop()
	s.relayerSessionsManager = s.setupNewRelayerSessionsManager()

	// Calculate when the claim window closes for this session
	claimWindowCloseHeight := sharedtypes.GetClaimWindowCloseHeight(&s.sharedParams, sessionEndHeight)
	// Move past the claim window close height
	s.advanceToBlock(claimWindowCloseHeight + 1)

	// Start the new relayer sessions manager
	err := s.relayerSessionsManager.Start(s.ctx)
	require.NoError(s.T(), err)
	waitSimulateIO()

	// Verify the session tree has been removed since the claim window was missed
	require.Len(s.T(), s.sessionTrees, 0)
	require.Equal(s.T(), 0, s.createClaimCallCount)
	require.Equal(s.T(), 0, s.submitProofCallCount)
}

// TestRestartAfterProofWindowClosed tests session persistence when the relayer is restarted
// after the proof window has closed.
// This verifies that the relayer correctly cleans up sessions after the proof window closes.
func (s *SessionPersistenceTestSuite) TestRestartAfterProofWindowClosed() {
	// Get the session end height from the active session header
	sessionEndHeight := s.activeSessionHeader.GetSessionEndBlockHeight()

	// Calculate when the claim window closes for this session
	claimWindowCloseHeight := sharedtypes.GetClaimWindowCloseHeight(&s.sharedParams, sessionEndHeight)
	// Move to one block before the claim window closes (a claim should be created)
	s.advanceToBlock(claimWindowCloseHeight - 1)

	// Verify the session tree exists and a claim has been created
	sessionTree := s.getActiveSessionTree()
	claimRoot := sessionTree.GetClaimRoot()
	require.NotNil(s.T(), claimRoot)
	require.Equal(s.T(), 1, s.createClaimCallCount)

	// Stop and recreate the relayer sessions manager
	s.relayerSessionsManager.Stop()
	s.relayerSessionsManager = s.setupNewRelayerSessionsManager()

	// Calculate when the proof window closes for this session
	proofWinodwCloseHeight := sharedtypes.GetProofWindowCloseHeight(&s.sharedParams, sessionEndHeight)
	// Move past the proof window close height
	s.advanceToBlock(proofWinodwCloseHeight + 1)

	// Start the new relayer sessions manager
	err := s.relayerSessionsManager.Start(s.ctx)
	require.NoError(s.T(), err)
	waitSimulateIO()

	// Verify the session tree has been removed since the proof window has closed
	require.Len(s.T(), s.sessionTrees, 0)
	// Verify no proofs were submitted since the proof window was already closed
	require.Equal(s.T(), 0, s.submitProofCallCount)
}

// getActiveSessionTree retrieves the current active session tree for testing purposes.
// It navigates through the session trees map structure to find the specific session tree
// for the active session header and supplier address.
func (s *SessionPersistenceTestSuite) getActiveSessionTree() relayer.SessionTree {
	// Extract session details from the active header
	sessionEndHeight := s.activeSessionHeader.GetSessionEndBlockHeight()
	sessionId := s.activeSessionHeader.GetSessionId()

	// Get the specific session tree for this supplier
	supplierSessionTrees, ok := s.sessionTrees[s.supplierOperatorAddress]
	require.True(s.T(), ok)

	// Get all session trees for this session end height
	sessionTreesWithEndHeight, ok := supplierSessionTrees[sessionEndHeight]
	require.True(s.T(), ok)

	// Get all session trees for this session ID
	sessionTree, ok := sessionTreesWithEndHeight[sessionId]
	require.True(s.T(), ok)

	return sessionTree
}

// setupNewRelayerSessionsManager creates and configures a new relayer sessions manager for testing.
// This is used both in the initial setup and when simulating restarts.
func (s *SessionPersistenceTestSuite) setupNewRelayerSessionsManager() relayer.RelayerSessionsManager {
	// Initialize a new session trees map
	s.sessionTrees = make(session.SessionsTreesMap)
	// Create an inspector that will monitor the session trees for testing
	sessionTreesInspector := session.WithSessionTreesInspector(&s.sessionTrees)

	// Create a new replay observable for blocks
	s.blocksObs, s.blockPublishCh = channel.NewReplayObservable[client.Block](s.ctx, 20)

	// Set up a listener to update the latest block whenever a new block comes in
	channel.ForEach(
		context.Background(),
		s.blocksObs,
		func(ctx context.Context, block client.Block) {
			s.latestBlock = block
		},
	)

	// Create a new relayer sessions manager with the configured dependencies
	relayerSessionsManager, err := session.NewRelayerSessions(s.deps, s.storesDirectoryOpt, sessionTreesInspector)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), relayerSessionsManager)

	// Insert the mined relays observable into the sessions manager
	relayerSessionsManager.InsertRelays(s.minedRelaysObs)

	return relayerSessionsManager
}

// setupSessionManagerDependencies configures all the mock dependencies needed
// by the relayer sessions manager for testing.
func (s *SessionPersistenceTestSuite) setupSessionManagerDependencies() depinject.Config {
	ctrl := gomock.NewController(s.T())

	// Set up all mock clients
	supplierClientMock := s.setupMockSupplierClient(ctrl)
	blockQueryClientMock := s.setupMockBlockQueryClient(ctrl)
	proofQueryClientMock := s.setupMockProofQueryClient(ctrl)
	s.blockClient = s.setupMockBlockClient(ctrl)

	// Create a new replay observable for blocks
	s.blocksObs, s.blockPublishCh = channel.NewReplayObservable[client.Block](s.ctx, 20)

	// Create supplier client map and add the mock supplier client
	supplierClientMap := supplier.NewSupplierClientMap()
	supplierClientMap.SupplierClients[s.supplierOperatorAddress] = supplierClientMock

	// Configure other required mock query clients
	sharedQueryClientMock := testqueryclients.NewTestSharedQueryClient(s.T())
	serviceQueryClientMock := testqueryclients.NewTestServiceQueryClient(s.T())
	bankQueryClient := testqueryclients.NewTestBankQueryClientWithBalance(s.T(), 1000000)

	// Create the dependency supply configuration
	deps := depinject.Supply(
		s.blockClient,
		blockQueryClientMock,
		supplierClientMap,
		sharedQueryClientMock,
		serviceQueryClientMock,
		proofQueryClientMock,
		bankQueryClient,
		s.logger,
	)

	return deps
}

// setupMockSupplierClient creates and configures a mock supplier client
// for testing claim and proof submissions
func (s *SessionPersistenceTestSuite) setupMockSupplierClient(ctrl *gomock.Controller) *mockclient.MockSupplierClient {
	// Configure mock supplier client
	supplierClientMock := mockclient.NewMockSupplierClient(ctrl)

	// Mock the OperatorAddress method to return the test supplier address
	supplierClientMock.EXPECT().
		OperatorAddress().
		Return(s.supplierOperatorAddress).
		AnyTimes()

	// Mock the CreateClaims method to track claim creation
	supplierClientMock.EXPECT().
		CreateClaims(
			gomock.Any(),
			gomock.Any(),
			gomock.AssignableToTypeOf(([]client.MsgCreateClaim)(nil)),
		).
		DoAndReturn(func(ctx context.Context, timeoutHeight int64, claimMsgs ...*prooftypes.MsgCreateClaim) error {
			require.Len(s.T(), claimMsgs, 1)
			s.claimToReturn = &prooftypes.Claim{
				SupplierOperatorAddress: s.supplierOperatorAddress,
				SessionHeader:           s.activeSessionHeader,
				RootHash:                claimMsgs[0].GetRootHash(),
			}
			s.createClaimCallCount++
			return nil
		}).
		AnyTimes()

	// Mock the SubmitProofs method to track proof submission
	supplierClientMock.EXPECT().
		SubmitProofs(
			gomock.Any(),
			gomock.Any(),
			gomock.AssignableToTypeOf(([]client.MsgSubmitProof)(nil)),
		).
		DoAndReturn(func(ctx context.Context, timeoutHeight int64, proofMsgs ...*prooftypes.MsgSubmitProof) error {
			require.Len(s.T(), proofMsgs, 1)
			s.submitProofCallCount++
			return nil
		}).
		AnyTimes()

	return supplierClientMock
}

// setupMockBlockQueryClient creates and configures a mock block query client
// for testing block retrieval
func (s *SessionPersistenceTestSuite) setupMockBlockQueryClient(ctrl *gomock.Controller) *mockclient.MockCometRPC {
	// Configure mock block query client
	blockQueryClientMock := mockclient.NewMockCometRPC(ctrl)
	blockQueryClientMock.EXPECT().
		Block(gomock.Any(), gomock.AssignableToTypeOf((*int64)(nil))).
		DoAndReturn(
			func(_ context.Context, height *int64) (*coretypes.ResultBlock, error) {
				return &coretypes.ResultBlock{
					BlockID: cmttypes.BlockID{Hash: s.emptyBlockHash},
					Block: &cmttypes.Block{
						Header: cmttypes.Header{Height: *height},
					},
				}, nil
			},
		).
		AnyTimes()

	return blockQueryClientMock
}

// setupMockProofQueryClient creates and configures a mock proof query client
// for testing proof and claim retrieval
func (s *SessionPersistenceTestSuite) setupMockProofQueryClient(ctrl *gomock.Controller) *mockclient.MockProofQueryClient {
	// Configure mock proof query client
	proofQueryClientMock := mockclient.NewMockProofQueryClient(ctrl)
	proofQueryClientMock.EXPECT().
		GetParams(gomock.Any()).
		Return(&s.proofParams, nil).
		AnyTimes()
	proofQueryClientMock.EXPECT().
		GetClaim(
			gomock.Eq(s.ctx),
			gomock.Eq(s.supplierOperatorAddress),
			gomock.Eq(s.activeSessionHeader.SessionId),
		).
		DoAndReturn(
			func(_ any, _ any, _ any) (*prooftypes.Claim, error) {
				if s.claimToReturn == nil {
					return nil, status.Error(codes.NotFound, "claim not found")
				}
				return s.claimToReturn, nil
			},
		).
		AnyTimes()

	return proofQueryClientMock
}

// setupMockBlockClient creates and configures a mock block client
// for testing block sequence management
func (s *SessionPersistenceTestSuite) setupMockBlockClient(ctrl *gomock.Controller) *mockclient.MockBlockClient {
	// Configure mock block client
	blockClientMock := mockclient.NewMockBlockClient(ctrl)

	// Mock the LastBlock method to return the current latest block
	blockClientMock.EXPECT().LastBlock(gomock.Any()).
		DoAndReturn(func(_ any) client.Block {
			return s.latestBlock
		}).AnyTimes()

	// Mock the CommittedBlocksSequence method to return the blocks observable
	blockClientMock.EXPECT().
		CommittedBlocksSequence(gomock.Any()).
		DoAndReturn(func(_ any) observable.Observable[client.Block] {
			return s.blocksObs
		}).
		AnyTimes()

	// Mock the Close method to close the block publish channel
	blockClientMock.EXPECT().
		Close().
		DoAndReturn(func() {
			close(s.blockPublishCh)
		}).
		AnyTimes()

	return blockClientMock
}

// advanceToBlock advances the test chain to the specified height by
// publishing new blocks until the target height is reached.
func (s *SessionPersistenceTestSuite) advanceToBlock(height int64) {
	// Get the current height
	currentHeight := int64(0)
	currentBlock := s.blockClient.LastBlock(s.ctx)
	if currentBlock != nil {
		currentHeight = currentBlock.Height()
	}

	// Publish blocks until we reach the target height.
	// A loop is used instead of publishing the target height directly to populate
	// the block sequence observable with all blocks in between.
	for currentHeight < height {
		s.blockPublishCh <- testblock.NewAnyTimesBlock(s.T(), s.emptyBlockHash, currentHeight+1)
		currentHeight++
	}

	// Wait for I/O operations to complete
	waitSimulateIO()
}

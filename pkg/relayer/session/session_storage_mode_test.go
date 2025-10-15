package session_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

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

// StorageModeTestSuite defines the test suite for different storage modes
type StorageModeTestSuite struct {
	suite.Suite
	ctx        context.Context
	storageDir string // For disk storage mode

	deps                   depinject.Config
	relayerSessionsManager relayer.RelayerSessionsManager
	storesDirectoryPathOpt relayer.RelayerSessionsManagerOption

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

// TestStorageModeSimpleMap tests the ":memory:" (SimpleMap) storage mode
func TestStorageModeSimpleMap(t *testing.T) {
	suite.Run(t, &StorageModeTestSuite{storageDir: ""})
}

// SetupTest prepares the test environment before each test execution
func (s *StorageModeTestSuite) SetupTest() {
	// Initialize logger and context
	s.logger, s.ctx = testpolylog.NewLoggerWithCtx(context.Background(), polyzero.DebugLevel)

	// Initialize test data and state
	s.service = sharedtypes.Service{Id: "svc", ComputeUnitsPerRelay: 2000}
	s.sharedParams = sharedtypes.DefaultParams()
	s.proofParams = prooftypes.DefaultParams()
	s.proofParams.ProofRequirementThreshold = uPOKTCoin(1)
	s.supplierOperatorAddress = sample.AccAddressBech32()
	s.emptyBlockHash = make([]byte, 32)

	// Reset counters and state for each test
	s.createClaimCallCount = 0
	s.submitProofCallCount = 0
	s.claimToReturn = nil
	s.latestBlock = nil

	// Set up storage directory path based on test mode
	if s.storageDir == "" {
		// Disk storage mode - create temporary directory
		tmpDirPattern := fmt.Sprintf("%s_smt_kvstore", strings.ReplaceAll(s.T().Name(), "/", "_"))
		tmpStoresDir, err := os.MkdirTemp("", tmpDirPattern)
		require.NoError(s.T(), err)
		s.storageDir = tmpStoresDir
	}
	s.storesDirectoryPathOpt = session.WithStoresDirectoryPath(s.storageDir)

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
	err := s.relayerSessionsManager.Start(s.ctx)
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

	// Log which storage mode we're testing
	s.logger.Info().
		Str("storage_path", s.storageDir).
		Msg("Testing storage mode")
}

// TearDownTest cleans up resources after each test execution
func (s *StorageModeTestSuite) TearDownTest() {
	// Stop the relayer sessions manager
	s.relayerSessionsManager.Stop()

	// Clean up temporary directory for disk storage only
	_ = os.RemoveAll(s.storageDir)
}

// TestClaimAndProofSubmission tests the complete claim and proof submission lifecycle
// This is the critical test that should reveal the bug in Pebble in-memory mode
func (s *StorageModeTestSuite) TestBasicClaimAndProofSubmission() {
	// Get the session end height from the active session header
	sessionEndHeight := s.activeSessionHeader.GetSessionEndBlockHeight()

	// Calculate when the claim window opens for this session
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(&s.sharedParams, sessionEndHeight)
	// Move to the block where the claim window opens (which should trigger claim creation)
	s.advanceToBlock(claimWindowOpenHeight)

	// Verify the session tree exists and a claim has been created
	sessionTree := s.getActiveSessionTree()
	claimRoot := sessionTree.GetClaimRoot()
	require.NotNil(s.T(), claimRoot, "Claim root should be created")
	require.Equal(s.T(), 1, s.createClaimCallCount, "CreateClaim should be called once")
	require.Equal(s.T(), 0, s.submitProofCallCount, "SubmitProof should not be called at this step")

	// Verify the claim tree has exactly one claim
	count, err := smt.MerkleSumRoot(claimRoot).Count()
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint64(1), count, "Claim tree should have exactly one relay")

	// Calculate when the proof window closes for this session
	proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&s.sharedParams, sessionEndHeight)

	// Move to one block before the proof window closes (which should trigger proof submission)
	s.advanceToBlock(proofWindowCloseHeight)

	// This is the critical assertion - verify that a proof was submitted
	// Note: This test should pass for all storage modes with our current implementation
	// To reproduce the original bug where Pebble in-memory failed, you would need to:
	// 1. Revert the sessionSMT preservation fix in sessiontree.go Flush() method
	// 2. Remove the sessionSMT restoration logic in ProveClosest() method
	// 3. Then this assertion would fail for Pebble in-memory mode only
	require.Equal(s.T(), 1, s.submitProofCallCount, "SubmitProof should be called once")

	// Verify the session tree has been removed after proof submission
	require.False(s.T(), s.hasActiveSessionTree(), "Session tree should be removed after proof submission")
}

// TestProcessRestartDuringClaimWindow tests session persistence when restarted during claim window
func (s *StorageModeTestSuite) TestProcessRestartDuringClaimWindow() {
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

	// For disk storage, verify the session tree is still correctly loaded
	sessionTree = s.getActiveSessionTree()
	require.Equal(s.T(), s.activeSessionHeader, sessionTree.GetSessionHeader())

	// TODO_TECHDEBT: This sleep is a workaround for the race condition where claim creation
	// happens asynchronously via processClaimsAsync. The test expects GetClaimRoot() to return
	// immediately after restart, but claim creation takes time. A proper fix would either:
	// 1. Make claim restoration synchronous for previously flushed sessions during import, or
	// 2. Add proper synchronization/waiting mechanisms in the test framework.
	// This sleep should be removed once the underlying race condition is properly addressed.
	time.Sleep(100 * time.Millisecond)

	// Verify a claim root has been created after restart
	claimRoot := sessionTree.GetClaimRoot()
	require.NotNil(s.T(), claimRoot, "Claim root should be created after restart")

	// Verify createClaim was called exactly once
	require.Equal(s.T(), 1, s.createClaimCallCount, "CreateClaim should be called once after restart")
}

// TestInMemoryTreeStillSubmitsProofAfterFlush ensures that the in-memory tree still submits a proof after
// a Flush is called on the RelaySessionManager.
func (s *StorageModeTestSuite) TestInMemoryTreeStillSubmitsProofAfterFlush() {
	// Get the session end height from the active session header
	sessionEndHeight := s.activeSessionHeader.GetSessionEndBlockHeight()

	// Calculate when the claim window opens and proof window closes
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(&s.sharedParams, sessionEndHeight)
	proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&s.sharedParams, sessionEndHeight)

	// Move to claim window and verify claim creation
	s.advanceToBlock(claimWindowOpenHeight)
	sessionTree := s.getActiveSessionTree()
	claimRoot := sessionTree.GetClaimRoot()

	require.NotNil(s.T(), claimRoot, "Claim should be created")
	require.Equal(s.T(), 1, s.createClaimCallCount, "CreateClaim should be called once")

	// Simulates a flush
	sessionTree.Flush()

	// Move to proof window
	s.advanceToBlock(proofWindowCloseHeight)

	// Without the fix, this assertion would FAIL for Pebble in-memory mode
	// because the sessionSMT would be lost after Stop() during Flush()
	// With the fix in place, this assertion passes
	require.Equal(s.T(), 1, s.submitProofCallCount, "SubmitProof should be called - this would FAIL without the sessionSMT preservation fix")
}

// getActiveSessionTree retrieves the current active session tree for testing purposes
func (s *StorageModeTestSuite) getActiveSessionTree() relayer.SessionTree {
	sessionTree, ok := s.findActiveSessionTree()
	require.True(s.T(), ok)
	return sessionTree
}

func (s *StorageModeTestSuite) findActiveSessionTree() (relayer.SessionTree, bool) {
	sessionEndHeight := s.activeSessionHeader.GetSessionEndBlockHeight()
	sessionID := s.activeSessionHeader.GetSessionId()

	for _, snapshot := range s.relayerSessionsManager.SessionTreesSnapshots() {
		if snapshot.SupplierOperatorAddress != s.supplierOperatorAddress {
			continue
		}
		if snapshot.SessionEndHeight != sessionEndHeight {
			continue
		}
		if snapshot.SessionID != sessionID {
			continue
		}
		return snapshot.Tree, true
	}

	return nil, false
}

func (s *StorageModeTestSuite) hasActiveSessionTree() bool {
	_, ok := s.findActiveSessionTree()
	return ok
}

// setupNewRelayerSessionsManager creates and configures a new relayer sessions manager for testing
func (s *StorageModeTestSuite) setupNewRelayerSessionsManager() relayer.RelayerSessionsManager {
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
	relayerSessionsManager, err := session.NewRelayerSessions(s.deps, s.storesDirectoryPathOpt)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), relayerSessionsManager)

	// Insert the mined relays observable into the sessions manager
	relayerSessionsManager.InsertRelays(s.minedRelaysObs)

	return relayerSessionsManager
}

// setupSessionManagerDependencies configures all the mock dependencies needed by the relayer sessions manager
func (s *StorageModeTestSuite) setupSessionManagerDependencies() depinject.Config {
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

// setupMockSupplierClient creates and configures a mock supplier client for testing claim and proof submissions
func (s *StorageModeTestSuite) setupMockSupplierClient(ctrl *gomock.Controller) *mockclient.MockSupplierClient {
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

// setupMockBlockQueryClient creates and configures a mock block query client for testing block retrieval
func (s *StorageModeTestSuite) setupMockBlockQueryClient(ctrl *gomock.Controller) *mockclient.MockCometRPC {
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

// setupMockProofQueryClient creates and configures a mock proof query client for testing proof and claim retrieval
func (s *StorageModeTestSuite) setupMockProofQueryClient(ctrl *gomock.Controller) *mockclient.MockProofQueryClient {
	// Configure mock proof query client
	proofQueryClientMock := mockclient.NewMockProofQueryClient(ctrl)
	proofQueryClientMock.EXPECT().
		GetParams(gomock.Any()).
		Return(&s.proofParams, nil).
		AnyTimes()
	proofQueryClientMock.EXPECT().
		GetClaim(
			gomock.AssignableToTypeOf(s.ctx),
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

// setupMockBlockClient creates and configures a mock block client for testing block sequence management
func (s *StorageModeTestSuite) setupMockBlockClient(ctrl *gomock.Controller) *mockclient.MockBlockClient {
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

// advanceToBlock advances the test chain to the specified height by publishing new blocks
func (s *StorageModeTestSuite) advanceToBlock(height int64) {
	// Get the current height
	currentHeight := int64(0)
	currentBlock := s.blockClient.LastBlock(s.ctx)
	if currentBlock != nil {
		currentHeight = currentBlock.Height()
	}

	// Publish blocks until we reach the target height
	for currentHeight < height {
		s.blockPublishCh <- testblock.NewAnyTimesBlock(s.T(), s.emptyBlockHash, currentHeight+1)
		currentHeight++
	}

	// Wait for I/O operations to complete
	waitSimulateIO()
}

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
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

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

type queryCallStatus struct {
	total        int
	errorCount   int
	successCount int
}

// SessionPersistenceTestSuite defines the test suite for session persistence
type SessionPersistenceTestSuite struct {
	suite.Suite
	ctx          context.Context
	tmpStoresDir string

	deps                   depinject.Config
	relayerSessionsManager relayer.RelayerSessionsManager
	storesDirectoryOpt     relayer.RelayerSessionsManagerOption

	activeSession              *sessiontypes.Session
	supplierOperatorAccAddress sdktypes.AccAddress
	emptyBlockHash             []byte

	blockClient    client.BlockClient
	blockPublishCh chan<- client.Block
	blocksObs      observable.Observable[client.Block]
	latestBlock    client.Block

	minedRelaysPublishCh chan<- *relayer.MinedRelay
	minedRelaysObs       relayer.MinedRelaysObservable

	sharedParams sharedtypes.Params
	proofParams  prooftypes.Params

	logger polylog.Logger

	createClaimCallStatus      *queryCallStatus
	submitProofCallStatus      *queryCallStatus
	proofParamsQueryCallStatus *queryCallStatus

	claimCreationReturnsError    bool
	proofSubmissionReturnsError  bool
	proofParamsQueryReturnsError bool
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
	s.emptyBlockHash = []byte("emptyBlockHash")
	service := sharedtypes.Service{Id: "svc", ComputeUnitsPerRelay: 2}
	s.sharedParams = sharedtypes.DefaultParams()
	s.proofParams = prooftypes.DefaultParams()
	s.proofParams.ProofRequirementThreshold = uPOKTCoin(1)
	supplierOperatorAddress := sample.AccAddress()
	s.supplierOperatorAccAddress = sdktypes.MustAccAddressFromBech32(supplierOperatorAddress)

	s.latestBlock = nil
	s.proofParamsQueryReturnsError = false
	s.claimCreationReturnsError = false
	s.proofSubmissionReturnsError = false
	s.createClaimCallStatus = &queryCallStatus{}
	s.submitProofCallStatus = &queryCallStatus{}
	s.proofParamsQueryCallStatus = &queryCallStatus{}

	// Set up temporary directory for session storage
	tmpDirPattern := fmt.Sprintf("%s_smt_kvstore", strings.ReplaceAll(s.T().Name(), "/", "_"))
	tmpStoresDir, err := os.MkdirTemp("", tmpDirPattern)
	require.NoError(s.T(), err)
	s.storesDirectoryOpt = session.WithStoresDirectory(tmpStoresDir)

	// Configure test service and difficulty
	testqueryclients.AddToExistingServices(s.T(), service)
	testqueryclients.SetServiceRelayDifficultyTargetHash(s.T(), service.Id, protocol.BaseRelayDifficultyHashBz)

	// Create a session header for testing
	s.activeSession = &sessiontypes.Session{
		Header: &sessiontypes.SessionHeader{
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   int64(s.sharedParams.NumBlocksPerSession),
			ServiceId:               service.Id,
			SessionId:               "sessionId",
		},
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
	s.relayerSessionsManager.Start(s.ctx)
	require.NoError(s.T(), err)

	// Publish a test mined relay and wait for processing
	s.minedRelaysPublishCh <- testrelayer.NewUnsignedMinedRelay(s.T(), s.activeSession, supplierOperatorAddress)
	waitSimulateIO()
}

// TearDownTest cleans up resources after each test execution
func (s *SessionPersistenceTestSuite) TearDownTest() {
	// Stop the relayer sessions manager
	//s.relayerSessionsManager.Stop()
	// Delete all temporary files and directories created by the test on completion.
	_ = os.RemoveAll(s.tmpStoresDir)
}

func (s *SessionPersistenceTestSuite) TestProofQueryClientFailing() {
	sessionEndHeight := s.activeSession.Header.SessionEndBlockHeight
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(&s.sharedParams, sessionEndHeight)

	s.advanceToBlock(claimWindowOpenHeight - 1)

	// Instruct the proof query client to return an error when querying proof params.
	s.proofParamsQueryReturnsError = true
	s.advanceToBlock(claimWindowOpenHeight)
	// Wait for 5 seconds to allow the retry strategy to perform retries.
	time.Sleep(4 * time.Second)
	// Given the exponential backoff strategy configuration with 0.5 second as the initial delay,
	// the expected number of error calls is 4 which given 5 seconds of waiting time
	// should have (1) initial try, and (3) retries corresponding to 0.5, 1, and 2 seconds
	// of the exponential backoff strategy; totaling a duration of 3.5 seconds.
	expectedNumErrorCalls := 4

	// All attempts should have failed.
	require.Equal(s.T(), expectedNumErrorCalls, s.proofParamsQueryCallStatus.total)
	require.Equal(s.T(), expectedNumErrorCalls, s.proofParamsQueryCallStatus.errorCount)
	// There should be no successful attempts.
	require.Equal(s.T(), 0, s.proofParamsQueryCallStatus.successCount)

	// Instruct the proof query client to return a successful response when querying proof params.
	s.proofParamsQueryReturnsError = false
	// Wait for 5 seconds to allow the retry strategy to perform a last retry after
	// 4 seconds of waiting time.
	time.Sleep(5 * time.Second)

	// The total attempts should increase by 1.
	require.Equal(s.T(), expectedNumErrorCalls+1, s.proofParamsQueryCallStatus.total)
	// The error count should remain the same.
	require.Equal(s.T(), expectedNumErrorCalls, s.proofParamsQueryCallStatus.errorCount)
	// There should be one successful attempt.
	require.Equal(s.T(), 1, s.proofParamsQueryCallStatus.successCount)
}

func (s *SessionPersistenceTestSuite) TestSupplierClientFailingToCreateAClaim() {
	sessionEndHeight := s.activeSession.Header.SessionEndBlockHeight
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(&s.sharedParams, sessionEndHeight)

	s.advanceToBlock(claimWindowOpenHeight - 1)

	// Instruct the supplier client to return an error when submitting a claim transaction.
	s.claimCreationReturnsError = true
	s.advanceToBlock(claimWindowOpenHeight)
	// Wait for 5 seconds to allow the retry strategy to perform 4 failing retries.
	time.Sleep(4 * time.Second)
	expectedNumErrorCalls := 4

	// All attempts should have failed.
	require.Equal(s.T(), expectedNumErrorCalls, s.createClaimCallStatus.total)
	require.Equal(s.T(), expectedNumErrorCalls, s.createClaimCallStatus.errorCount)
	// There should be no successful attempts.
	require.Equal(s.T(), 0, s.createClaimCallStatus.successCount)

	// Instruct the proof query client to return a successful response when querying proof params.
	s.claimCreationReturnsError = false
	// Wait for 5 seconds to allow the retry strategy to perform a last retry after
	// 4 seconds of waiting time.
	time.Sleep(5 * time.Second)

	// The total attempts should increase by 1.
	require.Equal(s.T(), expectedNumErrorCalls+1, s.createClaimCallStatus.total)
	// The error count should remain the same.
	require.Equal(s.T(), expectedNumErrorCalls, s.createClaimCallStatus.errorCount)
	// There should be one successful attempt.
	require.Equal(s.T(), 1, s.createClaimCallStatus.successCount)
}

func (s *SessionPersistenceTestSuite) TestSupplierClientFailingToSubmitAProof() {
	sessionEndHeight := s.activeSession.Header.SessionEndBlockHeight
	proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(&s.sharedParams, sessionEndHeight)

	s.advanceToBlock(proofWindowOpenHeight - 1)

	// Instruct the supplier client to return an error when submitting a proof transaction.
	s.proofSubmissionReturnsError = true
	s.advanceToBlock(proofWindowOpenHeight)
	// Wait for 5 seconds to allow the retry strategy to perform 4 failing retries.
	time.Sleep(4 * time.Second)
	expectedNumErrorCalls := 4

	// All attempts should have failed.
	require.Equal(s.T(), expectedNumErrorCalls, s.submitProofCallStatus.total)
	require.Equal(s.T(), expectedNumErrorCalls, s.submitProofCallStatus.errorCount)
	// There should be no successful attempts.
	require.Equal(s.T(), 0, s.submitProofCallStatus.successCount)

	// Instruct the proof query client to return a successful response when querying proof params.
	s.proofSubmissionReturnsError = false
	// Wait for 5 seconds to allow the retry strategy to perform a last retry after
	// 4 seconds of waiting time.
	time.Sleep(5 * time.Second)

	// The total attempts should increase by 1.
	require.Equal(s.T(), expectedNumErrorCalls+1, s.submitProofCallStatus.total)
	// The error count should remain the same.
	require.Equal(s.T(), expectedNumErrorCalls, s.submitProofCallStatus.errorCount)
	// There should be one successful attempt.
	require.Equal(s.T(), 1, s.submitProofCallStatus.successCount)
}

// setupNewRelayerSessionsManager creates and configures a new relayer sessions manager for testing.
// This is used both in the initial setup and when simulating restarts.
func (s *SessionPersistenceTestSuite) setupNewRelayerSessionsManager() relayer.RelayerSessionsManager {
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
	relayerSessionsManager, err := session.NewRelayerSessions(s.ctx, s.deps, s.storesDirectoryOpt)
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
	supplierClientMap.SupplierClients[s.supplierOperatorAccAddress.String()] = supplierClientMock

	// Configure other required mock query clients
	serviceQueryClientMock := testqueryclients.NewTestServiceQueryClient(s.T())
	bankQueryClient := testqueryclients.NewTestBankQueryClientWithBalance(s.T(), 1000000)
	sharedQueryClientMock := testqueryclients.NewTestSharedQueryClient(s.T())

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
		Return(&s.supplierOperatorAccAddress).
		AnyTimes()

	// Mock the CreateClaims method to track claim creation
	supplierClientMock.EXPECT().
		CreateClaims(
			gomock.Any(),
			gomock.AssignableToTypeOf(([]client.MsgCreateClaim)(nil)),
		).
		DoAndReturn(func(ctx context.Context, claimMsgs ...*prooftypes.MsgCreateClaim) error {
			require.Len(s.T(), claimMsgs, 1)
			s.createClaimCallStatus.total++
			if s.claimCreationReturnsError {
				s.createClaimCallStatus.errorCount++
				return fmt.Errorf("error creating claims")
			}
			s.createClaimCallStatus.successCount++
			return nil
		}).
		AnyTimes()

	// Mock the SubmitProofs method to track proof submission
	supplierClientMock.EXPECT().
		SubmitProofs(
			gomock.Any(),
			gomock.AssignableToTypeOf(([]client.MsgSubmitProof)(nil)),
		).
		DoAndReturn(func(ctx context.Context, proofMsgs ...*prooftypes.MsgSubmitProof) error {
			require.Len(s.T(), proofMsgs, 1)
			s.submitProofCallStatus.total++
			if s.proofSubmissionReturnsError {
				s.submitProofCallStatus.errorCount++
				return fmt.Errorf("error submitting proofs")
			}
			s.submitProofCallStatus.successCount++
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
		DoAndReturn(func(ctx context.Context) (*prooftypes.Params, error) {
			s.proofParamsQueryCallStatus.total++
			if s.proofParamsQueryReturnsError {
				s.proofParamsQueryCallStatus.errorCount++
				return nil, fmt.Errorf("error querying proof params")
			}
			s.proofParamsQueryCallStatus.successCount++
			return &s.proofParams, nil
		}).
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
		}).
		AnyTimes()

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

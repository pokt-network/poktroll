package query_test

import (
	"context"
	"net"
	"testing"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	querytypes "github.com/pokt-network/poktroll/pkg/client/query/types"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testclient/testqueryclients"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const numCalls = 4

// QueryCacheTestSuite runs all the tests for the query clients that cache their responses.
type QueryCacheTestSuite struct {
	suite.Suite

	queryClients *queryClients
	queryServers *queryServers

	listener       *bufconn.Listener
	grpcServer     *grpc.Server
	grpcClientConn *grpc.ClientConn
}

func (s *QueryCacheTestSuite) SetupTest() {
	ctx := context.Background()
	logger := polylog.Ctx(ctx)

	// Create the gRPC server for the query clients
	s.grpcServer, s.listener, s.queryServers = createGRPCServer(s.T())

	// Create a gRPC client connection to the gRPC server
	s.grpcClientConn = createGRPCClienConn(s.T(), s.listener)

	// Create a depinject.Config with the cache dependencies
	deps := supplyCacheDeps()

	// Create a new depinject config with a supplied gRPC client connection and logger
	// needed by the query clients.
	deps = depinject.Configs(deps, depinject.Supply(s.grpcClientConn, logger))

	// Create the query clients under test.
	s.queryClients = createQueryClients(s.T(), deps)
}

func (s *QueryCacheTestSuite) TearDownTest() {
	s.grpcServer.Stop()
}

func TestQueryClientCache(t *testing.T) {
	suite.Run(t, &QueryCacheTestSuite{})
}

func (s *QueryCacheTestSuite) TestKeyValueCache_ServiceQuerier() {
	ctx := context.Background()

	// Call the GetService method numCalls times and assert that the service server
	// is reached only once.
	for i := 0; i < numCalls; i++ {
		_, err := s.queryClients.service.GetService(ctx, "serviceId")
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.queryServers.service.ServiceCallCounter.CallCount())

	// Call the GetServiceRelayDifficulty method numCalls times and assert that the service
	// server is reached only once.
	for i := 0; i < numCalls; i++ {
		_, err := s.queryClients.service.GetServiceRelayDifficulty(ctx, "serviceId")
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.queryServers.service.RelayMiningDifficultyCallCounter.CallCount())
}

func (s *QueryCacheTestSuite) TestKeyValueCache_ApplicationQuerier() {
	ctx := context.Background()
	appAddress := sample.AccAddress()

	// Call the GetApplication method numCalls times and assert that the application server
	// is reached only once.
	for i := 0; i < numCalls; i++ {
		_, err := s.queryClients.application.GetApplication(ctx, appAddress)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.queryServers.application.AppCallCounter.CallCount())

	// Call the GetParams method numCalls times and assert that the application server
	// is reached only once.
	for i := 0; i < numCalls; i++ {
		_, err := s.queryClients.application.GetParams(ctx)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.queryServers.application.ParamsCallCounter.CallCount())
}

func (s *QueryCacheTestSuite) TestKeyValueCache_SupplierQuerier() {
	ctx := context.Background()
	supplierAddress := sample.AccAddress()

	// Call the GetSupplier method numCalls times and assert that the supplier server
	// is reached only once.
	for i := 0; i < numCalls; i++ {
		_, err := s.queryClients.supplier.GetSupplier(ctx, supplierAddress)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.queryServers.supplier.SupplierCallCounter.CallCount())
}

func (s *QueryCacheTestSuite) TestKeyValueCache_SessionQuerier() {
	ctx := context.Background()
	appAddress := sample.AccAddress()
	serviceId := "serviceId"
	blockHeight := int64(1)

	// Call the GetSession method numCalls times and assert that the session server
	// is reached only once.
	for i := 0; i < numCalls; i++ {
		_, err := s.queryClients.session.GetSession(ctx, appAddress, serviceId, blockHeight)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.queryServers.session.SessionCallCounter.CallCount())

	// Call the GetParams method numCalls times and assert that the session server
	// is reached only once.
	for i := 0; i < numCalls; i++ {
		_, err := s.queryClients.session.GetParams(ctx)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.queryServers.session.ParamsCallCounter.CallCount())
}

func (s *QueryCacheTestSuite) TestKeyValueCache_SharedQuerier() {
	ctx := context.Background()

	// Call the GetParams method numCalls times and assert that the shared server
	// is reached only once.
	for i := 0; i < numCalls; i++ {
		_, err := s.queryClients.shared.GetParams(ctx)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.queryServers.shared.ParamsCallCounter.CallCount())
}

func (s *QueryCacheTestSuite) TestKeyValueCache_ProofQuerier() {
	ctx := context.Background()

	// Call the GetParams method numCalls times and assert that the proof server
	// is reached only once.
	for i := 0; i < numCalls; i++ {
		_, err := s.queryClients.proof.GetParams(ctx)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.queryServers.proof.ParamsCallCounter.CallCount())
}

func (s *QueryCacheTestSuite) TestKeyValueCache_BankQuerier() {
	ctx := context.Background()
	accountAddress := sample.AccAddress()

	// Call the GetBalance method numCalls times and assert that the bank server
	// is reached only once.
	for i := 0; i < numCalls; i++ {
		_, err := s.queryClients.bank.GetBalance(ctx, accountAddress)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.queryServers.bank.BalanceCallCounter.CallCount())
}

func (s *QueryCacheTestSuite) TestKeyValueCache_AccountQuerier() {
	ctx := context.Background()
	accountAddress := sample.AccAddress()

	// Call the GetAccount method numCalls times and assert that the account server
	// is reached only once.
	for i := 0; i < numCalls; i++ {
		_, err := s.queryClients.account.GetAccount(ctx, accountAddress)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.queryServers.account.AccountCallCounter.CallCount())
}

// supplyCacheDeps supplies all the cache dependencies required by the query clients.
func supplyCacheDeps() depinject.Config {
	return depinject.Supply(
		cache.NewKeyValueCache[sharedtypes.Service](),
		cache.NewKeyValueCache[servicetypes.RelayMiningDifficulty](),
		cache.NewKeyValueCache[apptypes.Application](),
		cache.NewKeyValueCache[sharedtypes.Supplier](),
		cache.NewKeyValueCache[*sessiontypes.Session](),
		cache.NewKeyValueCache[querytypes.Balance](),
		cache.NewKeyValueCache[querytypes.BlockHash](),

		cache.NewParamsCache[sharedtypes.Params](),
		cache.NewParamsCache[apptypes.Params](),
		cache.NewParamsCache[sessiontypes.Params](),
		cache.NewParamsCache[prooftypes.Params](),

		cache.NewKeyValueCache[cosmostypes.AccountI](),
	)
}

// createQueryClients creates all the query clients that cache their responses
// and are being tested in this test suite.
func createQueryClients(t *testing.T, deps depinject.Config) *queryClients {
	var err error
	queryClients := &queryClients{}

	queryClients.service, err = query.NewServiceQuerier(deps)
	require.NoError(t, err)

	queryClients.application, err = query.NewApplicationQuerier(deps)
	require.NoError(t, err)

	queryClients.supplier, err = query.NewSupplierQuerier(deps)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	cometClientMock := mockclient.NewMockCometRPC(ctrl)

	deps = depinject.Configs(deps, depinject.Supply(cometClientMock))

	queryClients.shared, err = query.NewSharedQuerier(deps)
	require.NoError(t, err)

	// Supply the shared query client which the session query client depends on.
	deps = depinject.Configs(deps, depinject.Supply(queryClients.shared))
	queryClients.session, err = query.NewSessionQuerier(deps)
	require.NoError(t, err)

	queryClients.proof, err = query.NewProofQuerier(deps)
	require.NoError(t, err)

	queryClients.bank, err = query.NewBankQuerier(deps)
	require.NoError(t, err)

	queryClients.account, err = query.NewAccountQuerier(deps)
	require.NoError(t, err)

	return queryClients
}

// queryClients contains all the query clients that cache their responses and
// being tested in this test suite.
type queryClients struct {
	service     client.ServiceQueryClient
	application client.ApplicationQueryClient
	supplier    client.SupplierQueryClient
	session     client.SessionQueryClient
	shared      client.SharedQueryClient
	proof       client.ProofQueryClient

	bank    client.BankQueryClient
	account client.AccountQueryClient
}

// queryServers contains all the mock gRPC query servers that the query clients
// in the test suite are calling.
type queryServers struct {
	service     *testqueryclients.MockServiceQueryServer
	application *testqueryclients.MockApplicationQueryServer
	supplier    *testqueryclients.MockSupplierQueryServer
	session     *testqueryclients.MockSessionQueryServer
	shared      *testqueryclients.MockSharedQueryServer
	proof       *testqueryclients.MockProofQueryServer

	bank    *testqueryclients.MockBankQueryServer
	account *testqueryclients.MockAccountQueryServer
}

// createGRPCServer creates a gRPC server with all the mock query servers
// The gRPC server uses a bufconn.Listener to avoid port conflicts in concurrent tests.
func createGRPCServer(t *testing.T) (*grpc.Server, *bufconn.Listener, *queryServers) {
	// Create the gRPC server
	grpcServer := grpc.NewServer()
	listener := bufconn.Listen(1024 * 1024)
	queryServers := &queryServers{}

	// Register all the mock query servers used in the test with the gRPC server.

	queryServers.service = &testqueryclients.MockServiceQueryServer{}
	servicetypes.RegisterQueryServer(grpcServer, queryServers.service)

	queryServers.application = &testqueryclients.MockApplicationQueryServer{}
	apptypes.RegisterQueryServer(grpcServer, queryServers.application)

	queryServers.supplier = &testqueryclients.MockSupplierQueryServer{}
	suppliertypes.RegisterQueryServer(grpcServer, queryServers.supplier)

	queryServers.session = &testqueryclients.MockSessionQueryServer{}
	sessiontypes.RegisterQueryServer(grpcServer, queryServers.session)

	queryServers.shared = &testqueryclients.MockSharedQueryServer{}
	sharedtypes.RegisterQueryServer(grpcServer, queryServers.shared)

	queryServers.proof = &testqueryclients.MockProofQueryServer{}
	prooftypes.RegisterQueryServer(grpcServer, queryServers.proof)

	queryServers.bank = &testqueryclients.MockBankQueryServer{}
	banktypes.RegisterQueryServer(grpcServer, queryServers.bank)

	queryServers.account = &testqueryclients.MockAccountQueryServer{}
	authtypes.RegisterQueryServer(grpcServer, queryServers.account)

	// Start the gRPC server
	go func() {
		err := grpcServer.Serve(listener)
		require.NoError(t, err)
	}()

	return grpcServer, listener, queryServers
}

// createGRPCClienConn creates a gRPC client connection to the bufconn.Listener.
func createGRPCClienConn(t *testing.T, listener *bufconn.Listener) *grpc.ClientConn {
	dialer := func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}

	grpcClientConn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	return grpcClientConn
}

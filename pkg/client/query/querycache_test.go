package query_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	comettypes "github.com/cometbft/cometbft/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/grpc"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	querycache "github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const numCalls = 4

// QueryCacheTestSuite runs all the tests for the query clients that cache their responses.
type QueryCacheTestSuite struct {
	suite.Suite

	queryClients *queryClients
	rpcCallCount rpcCallCount
}

func (s *QueryCacheTestSuite) SetupTest() {
	s.rpcCallCount = rpcCallCount{}

	// Create a depinject.Config with the cache dependencies
	deps := supplyCacheDeps(s.T())

	// Create the query clients under test.
	s.queryClients = s.createQueryClients(s.T(), deps)
}

func TestQueryClientCache(t *testing.T) {
	suite.Run(t, &QueryCacheTestSuite{})
}

func (s *QueryCacheTestSuite) TestKeyValueCache_ServiceQuerier_Services() {
	ctx := context.Background()

	// Assert that the server has not been reached yet.
	require.Equal(s.T(), 0, s.rpcCallCount.services)

	// Call the GetService method numCalls times and assert that the server
	// is reached only once.
	for range numCalls {
		_, err := s.queryClients.service.GetService(ctx, "serviceId")
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.services)
}

func (s *QueryCacheTestSuite) TestKeyValueCache_ServiceQuerier_RelayMiningDifficulty() {
	ctx := context.Background()

	// Assert that the server has not been reached yet.
	require.Equal(s.T(), 0, s.rpcCallCount.difficulty)

	// Call the GetServiceRelayDifficulty method numCalls times and assert that the
	// server is reached only once.
	for range numCalls {
		_, err := s.queryClients.service.GetServiceRelayDifficulty(ctx, "serviceId")
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.difficulty)
}

func (s *QueryCacheTestSuite) TestKeyValueCache_ApplicationQuerier_Applications() {
	ctx := context.Background()
	appAddress := sample.AccAddress()

	// Assert that the server has not been reached yet.
	require.Equal(s.T(), 0, s.rpcCallCount.apps)

	// Call the GetApplication method numCalls times and assert that the server
	// is reached only once.
	for range numCalls {
		_, err := s.queryClients.application.GetApplication(ctx, appAddress)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.apps)
}

func (s *QueryCacheTestSuite) TestKeyValueCache_ApplicationQuerier_Params() {
	ctx := context.Background()

	// Assert that the server has not been reached yet.
	require.Equal(s.T(), 0, s.rpcCallCount.appParams)

	// Call the GetParams method numCalls times and assert that the server
	// is reached only once.
	for range numCalls {
		_, err := s.queryClients.application.GetParams(ctx)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.appParams)
}

func (s *QueryCacheTestSuite) TestKeyValueCache_SupplierQuerier_Suppliers() {
	ctx := context.Background()
	supplierAddress := sample.AccAddress()

	// Assert that the server has not been reached yet.
	require.Equal(s.T(), 0, s.rpcCallCount.suppliers)

	// Call the GetSupplier method numCalls times and assert that the server
	// is reached only once.
	for range numCalls {
		_, err := s.queryClients.supplier.GetSupplier(ctx, supplierAddress)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.suppliers)
}

func (s *QueryCacheTestSuite) TestKeyValueCache_SessionQuerier_Sessions() {
	ctx := context.Background()
	appAddress := sample.AccAddress()
	serviceId := "serviceId"
	blockHeight := int64(1)

	// Assert that the server has not been reached yet.
	require.Equal(s.T(), 0, s.rpcCallCount.sessions)

	// Call the GetSession method numCalls times and assert that the server
	// is reached only once.
	for range numCalls {
		_, err := s.queryClients.session.GetSession(ctx, appAddress, serviceId, blockHeight)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.sessions)
}

func (s *QueryCacheTestSuite) TestKeyValueCache_SessionQuerier_Params() {
	ctx := context.Background()

	// Assert that the server has not been reached yet.
	require.Equal(s.T(), 0, s.rpcCallCount.sessionParams)

	// Call the GetParams method numCalls times and assert that the server
	// is reached only once.
	for range numCalls {
		_, err := s.queryClients.session.GetParams(ctx)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.sessionParams)
}

func (s *QueryCacheTestSuite) TestKeyValueCache_SharedQuerier_Params() {
	ctx := context.Background()

	// Assert that the server has not been reached yet.
	require.Equal(s.T(), 0, s.rpcCallCount.sharedParams)
	require.Equal(s.T(), 0, s.rpcCallCount.blocks)

	// Call the GetParams method numCalls times and assert that the server
	// is reached only once.
	for range numCalls {
		_, err := s.queryClients.shared.GetParams(ctx)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.sharedParams)
	require.Equal(s.T(), 0, s.rpcCallCount.blocks)

	// Call the GetClaimWindowOpenHeight method numCalls times and assert that the server
	// is not reached again.
	for range numCalls {
		_, err := s.queryClients.shared.GetClaimWindowOpenHeight(ctx, 1)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.sharedParams)
	require.Equal(s.T(), 0, s.rpcCallCount.blocks)

	// Call the GetProofWindowOpenHeight method numCalls times and assert that the server
	// is not reached again.
	for range numCalls {
		_, err := s.queryClients.shared.GetProofWindowOpenHeight(ctx, 1)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.sharedParams)
	require.Equal(s.T(), 0, s.rpcCallCount.blocks)

	// Call the GetSessionGracePeriodEndHeight method numCalls times and assert that the server
	// is not reached again.
	for range numCalls {
		_, err := s.queryClients.shared.GetSessionGracePeriodEndHeight(ctx, 1)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.sharedParams)
	require.Equal(s.T(), 0, s.rpcCallCount.blocks)

	// Call the GetSessionBlockFrequency method numCalls times and assert that the server
	// is not reached again.
	for range numCalls {
		_, err := s.queryClients.shared.GetComputeUnitsToTokensMultiplier(ctx)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.sharedParams)
	require.Equal(s.T(), 0, s.rpcCallCount.blocks)

	supplierAddr := sample.AccAddress()

	// Call the GetEarliestSupplierClaimCommitHeight method numCalls times and assert that
	// the CometRPC server is reached only once.
	for range numCalls {
		_, err := s.queryClients.shared.GetEarliestSupplierClaimCommitHeight(ctx, 1, supplierAddr)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.sharedParams)
	require.Equal(s.T(), 1, s.rpcCallCount.blocks)

	// Call the GetEarliestSupplierProofCommitHeight method numCalls times and assert that
	// the CometRPC server is reached once again for a different block height
	for range numCalls {
		_, err := s.queryClients.shared.GetEarliestSupplierProofCommitHeight(ctx, 1, supplierAddr)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.sharedParams)
	require.Equal(s.T(), 2, s.rpcCallCount.blocks)
}

func (s *QueryCacheTestSuite) TestKeyValueCache_ProofQuerier_Params() {
	ctx := context.Background()

	// Assert that the server has not been reached yet.
	require.Equal(s.T(), 0, s.rpcCallCount.proofParams)

	// Call the GetParams method numCalls times and assert that the server
	// is reached only once.
	for range numCalls {
		_, err := s.queryClients.proof.GetParams(ctx)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.proofParams)
}

func (s *QueryCacheTestSuite) TestKeyValueCache_BankQuerier_Balances() {
	ctx := context.Background()
	accountAddress := sample.AccAddress()

	// Assert that the bank server has not been reached yet.
	require.Equal(s.T(), 0, s.rpcCallCount.balances)

	// Call the GetBalance method numCalls times and assert that the server
	// is reached only once.
	for range numCalls {
		_, err := s.queryClients.balance.GetBalance(ctx, accountAddress)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.balances)
}

func (s *QueryCacheTestSuite) TestKeyValueCache_AccountQuerier_Accounts() {
	ctx := context.Background()
	accountAddress := sample.AccAddress()

	// Assert that the account server has not been reached yet.
	require.Equal(s.T(), 0, s.rpcCallCount.accounts)

	// Call the GetAccount method numCalls times and assert that the server
	// is reached only once.
	for range numCalls {
		_, err := s.queryClients.account.GetAccount(ctx, accountAddress)
		require.NoError(s.T(), err)
	}
	require.Equal(s.T(), 1, s.rpcCallCount.accounts)
}

// createQueryClients creates all the query clients that cache their responses
// and are being tested in this test suite.
func (s *QueryCacheTestSuite) createQueryClients(t *testing.T, deps depinject.Config) *queryClients {
	var err error
	queryClients := &queryClients{}

	ctx := context.Background()
	logger := polylog.Ctx(ctx)

	// Create the CometRPC and GRPCClientConn mocks.
	cometClientMock := s.NewCometRPC()
	grpcClientConn := s.NewGRPCClientConn()

	deps = depinject.Configs(deps, depinject.Supply(cometClientMock, logger, grpcClientConn))

	queryClients.service, err = query.NewServiceQuerier(deps)
	require.NoError(t, err)

	queryClients.application, err = query.NewApplicationQuerier(deps)
	require.NoError(t, err)

	queryClients.supplier, err = query.NewSupplierQuerier(deps)
	require.NoError(t, err)

	queryClients.shared, err = query.NewSharedQuerier(deps)
	require.NoError(t, err)

	// Supply the shared query client which the session query client depends on.
	deps = depinject.Configs(deps, depinject.Supply(queryClients.shared))
	queryClients.session, err = query.NewSessionQuerier(deps)
	require.NoError(t, err)

	queryClients.proof, err = query.NewProofQuerier(deps)
	require.NoError(t, err)

	queryClients.balance, err = query.NewBankQuerier(deps)
	require.NoError(t, err)

	queryClients.account, err = query.NewAccountQuerier(deps)
	require.NoError(t, err)

	return queryClients
}

func (s *QueryCacheTestSuite) NewCometRPC() *mockclient.MockCometRPC {
	ctrl := gomock.NewController(s.T())
	cometClientMock := mockclient.NewMockCometRPC(ctrl)

	// Mock the Block method of the CometRPC client.
	cometClientMock.EXPECT().Block(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, height *int64) (*coretypes.ResultBlock, error) {
			// Increment the call count each time the Block method is called.
			s.rpcCallCount.blocks++

			return &coretypes.ResultBlock{
				Block: &comettypes.Block{
					Header: comettypes.Header{
						Height: *height,
					},
				},
				BlockID: comettypes.BlockID{
					Hash: []byte("test_hash"),
				},
			}, nil
		}).AnyTimes()

	return cometClientMock
}

func (s *QueryCacheTestSuite) NewGRPCClientConn() grpc.ClientConn {
	ctrl := gomock.NewController(s.T())
	grpcClientConn := mockclient.NewMockClientConn(ctrl)

	// Mock the Invoke method of the GRPCClientConn.
	// This method needs to return a valid shared params response.
	grpcClientConn.EXPECT().Invoke(
		gomock.Any(), // ctx
		"/poktroll.shared.Query/Params",
		gomock.Any(),
		gomock.Any(),
	).
		Do(func(_ context.Context, _ string, _ any, reply any, _ ...any) {
			// Increment the call count each time the Invoke method is called.
			s.rpcCallCount.sharedParams++

			// Return the default shared params.
			params := sharedtypes.DefaultParams()

			response, ok := reply.(*sharedtypes.QueryParamsResponse)
			require.True(s.T(), ok)

			response.Params = params
		}).AnyTimes()

	// Mock the Invoke method of the GRPCClientConn.
	// This method needs to return a valid codec.Any response that will be unmarshalled
	// into an account by the account querier.
	grpcClientConn.EXPECT().Invoke(
		gomock.Any(), // ctx
		"/cosmos.auth.v1beta1.Query/Account",
		gomock.Any(),
		gomock.Any(),
	).
		Do(func(_ context.Context, _ string, _ any, reply any, _ ...any) {
			// Increment the call count each time the Invoke method is called.
			s.rpcCallCount.accounts++

			// Create a base account with a public key.
			pubKey := secp256k1.GenPrivKey().PubKey()
			account := &authtypes.BaseAccount{}
			err := account.SetPubKey(pubKey)
			require.NoError(s.T(), err)

			// Create a codec.Any with the account.
			accountAny, err := codectypes.NewAnyWithValue(account)
			require.NoError(s.T(), err)

			response, ok := reply.(*authtypes.QueryAccountResponse)
			require.True(s.T(), ok)

			response.Account = accountAny
		}).AnyTimes()

	// Mock the Invoke method of the GRPCClientConn.
	// DEV_NOTE: This mock needs to be the last one to give the other mocks a chance
	// to catch their respective calls.
	grpcClientConn.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, method string, _ any, _ any, _ ...any) {
			// Increment the corresponding call count each time the Invoke method is called.
			switch method {
			case "/poktroll.application.Query/Params":
				s.rpcCallCount.appParams++
			case "/poktroll.proof.Query/Params":
				s.rpcCallCount.proofParams++
			case "/poktroll.session.Query/Params":
				s.rpcCallCount.sessionParams++
			case "/poktroll.service.Query/Service":
				s.rpcCallCount.services++
			case "/poktroll.service.Query/RelayMiningDifficulty":
				s.rpcCallCount.difficulty++
			case "/poktroll.supplier.Query/Supplier":
				s.rpcCallCount.suppliers++
			case "/poktroll.application.Query/Application":
				s.rpcCallCount.apps++
			case "/poktroll.session.Query/GetSession":
				s.rpcCallCount.sessions++
			case "/cosmos.bank.v1beta1.Query/Balance":
				s.rpcCallCount.balances++
			default:
				require.Failf(s.T(), "unexpected method: %s", method)
			}

		}).AnyTimes()

	return grpcClientConn
}

// rpcCallCount is a struct that keeps track of the number of times each RPC method is called.
type rpcCallCount struct {
	// poktroll key value calls
	services   int
	difficulty int
	apps       int
	suppliers  int
	sessions   int

	// poktroll params calls
	appParams     int
	sessionParams int
	sharedParams  int
	proofParams   int

	// cosmos-sdk calls
	blocks   int
	balances int
	accounts int
}

// queryClients contains all the query clients that cache their responses and
// being tested in this test suite.
type queryClients struct {
	service     client.ServiceQueryClient
	application client.ApplicationQueryClient
	supplier    client.SupplierQueryClient
	session     client.SessionQueryClient
	balance     client.BankQueryClient
	shared      client.SharedQueryClient
	proof       client.ProofQueryClient

	account client.AccountQueryClient
}

// supplyCacheDeps supplies all the cache dependencies required by the query clients.
func supplyCacheDeps(t *testing.T) depinject.Config {
	opts := memory.WithTTL(time.Minute)

	serviceCache, err := memory.NewKeyValueCache[sharedtypes.Service](opts)
	require.NoError(t, err)

	difficultyCache, err := memory.NewKeyValueCache[servicetypes.RelayMiningDifficulty](opts)
	require.NoError(t, err)

	appCache, err := memory.NewKeyValueCache[apptypes.Application](opts)
	require.NoError(t, err)

	supplierCache, err := memory.NewKeyValueCache[sharedtypes.Supplier](opts)
	require.NoError(t, err)

	sessionCache, err := memory.NewKeyValueCache[*sessiontypes.Session](opts)
	require.NoError(t, err)

	balanceCache, err := memory.NewKeyValueCache[query.Balance](opts)
	require.NoError(t, err)

	blockHashCache, err := memory.NewKeyValueCache[query.BlockHash](opts)
	require.NoError(t, err)

	claimsCache, err := memory.NewKeyValueCache[prooftypes.Claim](opts)
	require.NoError(t, err)

	sharedParamsCache, err := querycache.NewParamsCache[sharedtypes.Params](opts)
	require.NoError(t, err)

	appParamsCache, err := querycache.NewParamsCache[apptypes.Params](opts)
	require.NoError(t, err)

	sessionParamsCache, err := querycache.NewParamsCache[sessiontypes.Params](opts)
	require.NoError(t, err)

	proofParamsCache, err := querycache.NewParamsCache[prooftypes.Params](opts)
	require.NoError(t, err)

	accountCache, err := memory.NewKeyValueCache[cosmostypes.AccountI](opts)
	require.NoError(t, err)

	return depinject.Supply(
		serviceCache,
		difficultyCache,
		appCache,
		supplierCache,
		sessionCache,
		balanceCache,
		blockHashCache,
		claimsCache,

		sharedParamsCache,
		appParamsCache,
		sessionParamsCache,
		proofParamsCache,

		accountCache,
	)
}

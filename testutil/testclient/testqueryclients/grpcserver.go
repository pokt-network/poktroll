package testqueryclients

import (
	"context"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// callCounter is a simple struct that keeps track of the number of times a method is called
type callCounter struct {
	callCount int
}

func (c *callCounter) CallCount() int {
	return c.callCount
}

func (c *callCounter) Increment() {
	c.callCount++
}

// MockServiceQueryServer is a mock implementation of the servicetypes.QueryServer interface
// that keeps track of the number of times each method is called.
type MockServiceQueryServer struct {
	servicetypes.UnimplementedQueryServer
	ServiceCallCounter               callCounter
	RelayMiningDifficultyCallCounter callCounter
}

func (m *MockServiceQueryServer) Service(ctx context.Context, req *servicetypes.QueryGetServiceRequest) (*servicetypes.QueryGetServiceResponse, error) {
	m.ServiceCallCounter.Increment()
	return &servicetypes.QueryGetServiceResponse{}, nil
}

func (m *MockServiceQueryServer) RelayMiningDifficulty(ctx context.Context, req *servicetypes.QueryGetRelayMiningDifficultyRequest) (*servicetypes.QueryGetRelayMiningDifficultyResponse, error) {
	m.RelayMiningDifficultyCallCounter.Increment()
	return &servicetypes.QueryGetRelayMiningDifficultyResponse{}, nil
}

// MockApplicationQueryServer is a mock implementation of the apptypes.QueryServer interface
// that keeps track of the number of times each method is called.
type MockApplicationQueryServer struct {
	apptypes.UnimplementedQueryServer
	AppCallCounter    callCounter
	ParamsCallCounter callCounter
}

func (m *MockApplicationQueryServer) Application(ctx context.Context, req *apptypes.QueryGetApplicationRequest) (*apptypes.QueryGetApplicationResponse, error) {
	m.AppCallCounter.Increment()
	return &apptypes.QueryGetApplicationResponse{}, nil
}

func (m *MockApplicationQueryServer) Params(ctx context.Context, req *apptypes.QueryParamsRequest) (*apptypes.QueryParamsResponse, error) {
	m.ParamsCallCounter.Increment()
	return &apptypes.QueryParamsResponse{}, nil
}

// MockSupplierQueryServer is a mock implementation of the suppliertypes.QueryServer interface
// that keeps track of the number of times each method is called.
type MockSupplierQueryServer struct {
	suppliertypes.UnimplementedQueryServer
	SupplierCallCounter callCounter
}

func (m *MockSupplierQueryServer) Supplier(ctx context.Context, req *suppliertypes.QueryGetSupplierRequest) (*suppliertypes.QueryGetSupplierResponse, error) {
	m.SupplierCallCounter.Increment()
	return &suppliertypes.QueryGetSupplierResponse{}, nil
}

// MockSessionQueryServer is a mock implementation of the sessiontypes.QueryServer interface
// that keeps track of the number of times each method is called.
type MockSessionQueryServer struct {
	sessiontypes.UnimplementedQueryServer
	SessionCallCounter callCounter
	ParamsCallCounter  callCounter
}

func (m *MockSessionQueryServer) GetSession(ctx context.Context, req *sessiontypes.QueryGetSessionRequest) (*sessiontypes.QueryGetSessionResponse, error) {
	m.SessionCallCounter.Increment()
	return &sessiontypes.QueryGetSessionResponse{
		Session: &sessiontypes.Session{},
	}, nil
}

func (m *MockSessionQueryServer) Params(ctx context.Context, req *sessiontypes.QueryParamsRequest) (*sessiontypes.QueryParamsResponse, error) {
	m.ParamsCallCounter.Increment()
	return &sessiontypes.QueryParamsResponse{}, nil
}

// MockSharedQueryServer is a mock implementation of the sharedtypes.QueryServer interface
// that keeps track of the number of times each method is called.
type MockSharedQueryServer struct {
	sharedtypes.UnimplementedQueryServer
	ParamsCallCounter callCounter
}

func (m *MockSharedQueryServer) Params(ctx context.Context, req *sharedtypes.QueryParamsRequest) (*sharedtypes.QueryParamsResponse, error) {
	m.ParamsCallCounter.Increment()
	return &sharedtypes.QueryParamsResponse{
		Params: sharedtypes.Params{
			NumBlocksPerSession: 10,
		},
	}, nil
}

// MockProofQueryServer is a mock implementation of the prooftypes.QueryServer interface
// that keeps track of the number of times each method is called.
type MockProofQueryServer struct {
	prooftypes.UnimplementedQueryServer
	ParamsCallCounter callCounter
}

func (m *MockProofQueryServer) Params(ctx context.Context, req *prooftypes.QueryParamsRequest) (*prooftypes.QueryParamsResponse, error) {
	m.ParamsCallCounter.Increment()
	return &prooftypes.QueryParamsResponse{}, nil
}

// MockBankQueryServer is a mock implementation of the banktypes.QueryServer interface
// that keeps track of the number of times each method is called.
type MockBankQueryServer struct {
	banktypes.UnimplementedQueryServer
	BalanceCallCounter callCounter
}

func (m *MockBankQueryServer) Balance(ctx context.Context, req *banktypes.QueryBalanceRequest) (*banktypes.QueryBalanceResponse, error) {
	m.BalanceCallCounter.Increment()
	return &banktypes.QueryBalanceResponse{
		Balance: &cosmostypes.Coin{},
	}, nil
}

// MockAccountQueryServer is a mock implementation of the authtypes.QueryServer interface
// that keeps track of the number of times each method is called.
type MockAccountQueryServer struct {
	authtypes.UnimplementedQueryServer
	AccountCallCounter callCounter
}

func (m *MockAccountQueryServer) Account(ctx context.Context, req *authtypes.QueryAccountRequest) (*authtypes.QueryAccountResponse, error) {
	m.AccountCallCounter.Increment()
	pubKey := secp256k1.GenPrivKey().PubKey()

	account := &authtypes.BaseAccount{}
	account.SetPubKey(pubKey)
	accountAny, err := codectypes.NewAnyWithValue(account)
	if err != nil {
		return nil, err
	}

	return &authtypes.QueryAccountResponse{
		Account: accountAny,
	}, nil
}

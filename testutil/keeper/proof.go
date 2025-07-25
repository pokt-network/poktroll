package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil/integration"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/testutil/proof/mocks"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaykeeper "github.com/pokt-network/poktroll/x/gateway/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	supplierkeeper "github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// ProofModuleKeepers is an aggregation of the proof keeper and all its dependency
// keepers, and the codec that they share. Each keeper is embedded such that the
// ProofModuleKeepers implements all the interfaces of the keepers.
// To call a method which is common to multiple keepers (e.g. `#SetParams()`),
// the field corresponding to the desired keeper on which to call the method
// MUST be specified (e.g. `keepers.AccountKeeper#SetParams()`).
type ProofModuleKeepers struct {
	*keeper.Keeper
	prooftypes.BankKeeper
	prooftypes.SessionKeeper
	prooftypes.SupplierKeeper
	prooftypes.ApplicationKeeper
	prooftypes.AccountKeeper
	prooftypes.SharedKeeper
	prooftypes.ServiceKeeper

	Codec *codec.ProtoCodec
}

// ProofKeepersOpt is a function which receives and potentially modifies the context
// and proof keepers during construction of the aggregation.
type ProofKeepersOpt func(context.Context, *ProofModuleKeepers) context.Context

// ProofKeeper is a helper function to create a proof keeper and a context. It uses
// mocked dependencies only.
func ProofKeeper(t testing.TB) (keeper.Keeper, context.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(prooftypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	ctrl := gomock.NewController(t)
	mockBankKeeper := mocks.NewMockBankKeeper(ctrl)
	mockBankKeeper.EXPECT().
		SpendableCoins(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
				mapMu.RLock()
				defer mapMu.RUnlock()
				if coins, ok := mapAccAddrCoins[addr.String()]; ok {
					return coins
				}
				return sdk.Coins{}
			},
		).AnyTimes()

	mockSessionKeeper := mocks.NewMockSessionKeeper(ctrl)
	mockAppKeeper := mocks.NewMockApplicationKeeper(ctrl)
	mockAccountKeeper := mocks.NewMockAccountKeeper(ctrl)
	mockSharedKeeper := mocks.NewMockSharedKeeper(ctrl)
	mockServiceKeeper := mocks.NewMockServiceKeeper(ctrl)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockBankKeeper,
		mockSessionKeeper,
		mockAppKeeper,
		mockAccountKeeper,
		mockSharedKeeper,
		mockServiceKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	return k, ctx
}

// NewProofModuleKeepers is a helper function to create a proof keeper and a context. It uses
// real dependencies for all keepers except the bank keeper, which is mocked as it's not used
// directly by the proof keeper or its dependencies.
func NewProofModuleKeepers(t testing.TB, opts ...ProofKeepersOpt) (_ *ProofModuleKeepers, ctx context.Context) {
	t.Helper()

	// Collect store keys for all keepers which be constructed & interact with the state store.
	keys := storetypes.NewKVStoreKeys(
		prooftypes.StoreKey,
		banktypes.StoreKey,
		sessiontypes.StoreKey,
		suppliertypes.StoreKey,
		apptypes.StoreKey,
		gatewaytypes.StoreKey,
		authtypes.StoreKey,
		sharedtypes.StoreKey,
		servicetypes.StoreKey,
	)

	// Construct a multistore & mount store keys for each keeper that will interact with the state store.
	stateStore := integration.CreateMultiStore(keys, log.NewNopLogger())

	logger := log.NewTestLogger(t)
	ctx = sdk.NewContext(stateStore, cmtproto.Header{}, false, logger)

	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Construct a real account keeper so that public keys can be queried.
	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		map[string][]string{
			minttypes.ModuleName:     {authtypes.Minter},
			suppliertypes.ModuleName: {authtypes.Minter, authtypes.Burner, authtypes.Staking},
			prooftypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		},
		addresscodec.NewBech32Codec(app.AccountAddressPrefix),
		app.AccountAddressPrefix,
		authority.String(),
	)

	// Prepare the bank keeper
	blockedAddresses := map[string]bool{
		accountKeeper.GetAuthority(): false,
	}
	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		accountKeeper,
		blockedAddresses,
		authority.String(),
		logger,
	)

	// Construct a real shared keeper.
	sharedKeeper := sharedkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[sharedtypes.StoreKey]),
		logger,
		authority.String(),
	)
	require.NoError(t, sharedKeeper.SetParams(ctx, sharedtypes.DefaultParams()))

	// Construct gateway keeper with a mocked bank keeper.
	gatewayKeeper := gatewaykeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[gatewaytypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
		sharedKeeper,
	)
	require.NoError(t, gatewayKeeper.SetParams(ctx, gatewaytypes.DefaultParams()))

	// Construct an application keeper to add apps to sessions.
	appKeeper := appkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[apptypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
		accountKeeper,
		gatewayKeeper,
		sharedKeeper,
	)
	require.NoError(t, appKeeper.SetParams(ctx, apptypes.DefaultParams()))

	// Construct a service keeper need by the supplier keeper.
	serviceKeeper := servicekeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[servicetypes.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		bankKeeper,
	)

	// Construct a real supplier keeper to add suppliers to sessions.
	supplierKeeper := supplierkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[suppliertypes.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		bankKeeper,
		sharedKeeper,
		serviceKeeper,
	)
	require.NoError(t, supplierKeeper.SetParams(ctx, suppliertypes.DefaultParams()))

	// Construct a real session keeper so that sessions can be queried.
	sessionKeeper := sessionkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[sessiontypes.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		accountKeeper,
		bankKeeper,
		appKeeper,
		supplierKeeper,
		sharedKeeper,
	)
	require.NoError(t, sessionKeeper.SetParams(ctx, sessiontypes.DefaultParams()))

	// Construct a real proof keeper so that claims & proofs can be created.
	proofKeeper := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(keys[prooftypes.StoreKey]),
		log.NewNopLogger(),
		authority.String(),
		bankKeeper,
		sessionKeeper,
		appKeeper,
		accountKeeper,
		sharedKeeper,
		serviceKeeper,
	)
	require.NoError(t, proofKeeper.SetParams(ctx, prooftypes.DefaultParams()))

	keepers := &ProofModuleKeepers{
		Keeper:            &proofKeeper,
		BankKeeper:        &bankKeeper,
		SessionKeeper:     &sessionKeeper,
		SupplierKeeper:    &supplierKeeper,
		ApplicationKeeper: &appKeeper,
		AccountKeeper:     &accountKeeper,
		SharedKeeper:      &sharedKeeper,
		ServiceKeeper:     &serviceKeeper,

		Codec: cdc,
	}

	moduleBaseMint := sdk.NewCoins(sdk.NewCoin("upokt", math.NewInt(690000000000000042)))
	err := bankKeeper.MintCoins(ctx, suppliertypes.ModuleName, moduleBaseMint)
	require.NoError(t, err)

	// Apply any options to update the keepers or context prior to returning them.
	for _, opt := range opts {
		ctx = opt(ctx, keepers)
	}

	return keepers, ctx
}

// AddServiceActors adds a supplier and an application for a specific
// service so a successful session can be generated for testing purposes.
func (keepers *ProofModuleKeepers) AddServiceActors(
	ctx context.Context,
	t *testing.T,
	service *sharedtypes.Service,
	supplierOperatorAddr string,
	appAddr string,
) {
	t.Helper()

	supplierServices := []*sharedtypes.SupplierServiceConfig{
		{ServiceId: service.Id},
	}
	serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(supplierOperatorAddr, supplierServices, 1, 0)
	keepers.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{
		OperatorAddress:      supplierOperatorAddr,
		Services:             supplierServices,
		ServiceConfigHistory: serviceConfigHistory,
	})

	keepers.SetApplication(ctx, apptypes.Application{
		Address: appAddr,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			{ServiceId: service.Id},
		},
	})

	keepers.SetService(ctx, *service)
}

// GetSessionHeader is a helper to retrieve the session header
// for a specific (app, service, height).
func (keepers *ProofModuleKeepers) GetSessionHeader(
	ctx context.Context,
	t *testing.T,
	appAddr string,
	service *sharedtypes.Service,
	blockHeight int64,
) *sessiontypes.SessionHeader {
	t.Helper()

	sessionRes, err := keepers.GetSession(
		ctx,
		&sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			ServiceId:          service.Id,
			BlockHeight:        blockHeight,
		},
	)
	require.NoError(t, err)

	return sessionRes.GetSession().GetHeader()
}

// WithBlockHash sets the initial block hash for the context and returns the updated context.
func WithBlockHash(hash []byte) ProofKeepersOpt {
	return func(ctx context.Context, _ *ProofModuleKeepers) context.Context {
		return SetBlockHash(ctx, hash)
	}
}

// SetBlockHash updates the block hash for the given context and returns the updated context.
func SetBlockHash(ctx context.Context, hash []byte) context.Context {
	return sdk.UnwrapSDKContext(ctx).WithHeaderHash(hash)
}

// WithBlockHeight sets the initial block height for the context and returns the updated context.
func WithBlockHeight(height int64) ProofKeepersOpt {
	return func(ctx context.Context, _ *ProofModuleKeepers) context.Context {
		return SetBlockHeight(ctx, height)
	}
}

// SetBlockHeight updates the block height for the given context and returns the updated context.
func SetBlockHeight(ctx context.Context, height int64) context.Context {
	return sdk.UnwrapSDKContext(ctx).WithBlockHeight(height)
}

// GetBlockHeight returns the current block height for the given context.
func GetBlockHeight(ctx context.Context) int64 {
	return sdk.UnwrapSDKContext(ctx).BlockHeight()
}

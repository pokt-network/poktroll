package integration_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	pooltypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	integration "github.com/pokt-network/poktroll/testutil/integration"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	application "github.com/pokt-network/poktroll/x/application/module"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaykeeper "github.com/pokt-network/poktroll/x/gateway/keeper"
	gateway "github.com/pokt-network/poktroll/x/gateway/module"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	proofkeeper "github.com/pokt-network/poktroll/x/proof/keeper"
	proof "github.com/pokt-network/poktroll/x/proof/module"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	shared "github.com/pokt-network/poktroll/x/shared/module"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tokenomics "github.com/pokt-network/poktroll/x/tokenomics/module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func init() {
	cmd.InitSDKConfig()
}

// Example shows how to use the integration test framework to test the integration of SDK modules.
// Panics are used in this example, but in a real test case, you should use the testing.T object and assertions.
func TestExample(t *testing.T) {

	// Register the codec for all the interfacesPrepare all the interfaces
	registry := codectypes.NewInterfaceRegistry()
	// minttypes.RegisterInterfaces(registry)
	tokenomicstypes.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	gatewaytypes.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	sessiontypes.RegisterInterfaces(registry)
	apptypes.RegisterInterfaces(registry)
	suppliertypes.RegisterInterfaces(registry)
	prooftypes.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)

	// Prepare all the store keys
	storeKeys := storetypes.NewKVStoreKeys(
		tokenomicstypes.StoreKey,
		banktypes.StoreKey,
		gatewaytypes.StoreKey,
		sessiontypes.StoreKey,
		apptypes.StoreKey,
		suppliertypes.StoreKey,
		prooftypes.StoreKey,
		authtypes.StoreKey,
		minttypes.StoreKey,
		stakingtypes.StoreKey)

	// Prepare the context
	// logger := log.NewTestLogger(t)
	logger := log.NewNopLogger()
	cms := integration.CreateMultiStore(storeKeys, logger)
	newCtx := sdk.NewContext(cms, cmtproto.Header{}, true, logger)

	// Get the authority address
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Prepare the account keeper
	addrCodec := addresscodec.NewBech32Codec(app.AccountAddressPrefix)
	macPerms := map[string][]string{
		pooltypes.ModuleName:           {},
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		banktypes.ModuleName:           {authtypes.Minter, authtypes.Burner},
		tokenomicstypes.ModuleName:     {authtypes.Minter, authtypes.Burner},
		gatewaytypes.ModuleName:        {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		sessiontypes.ModuleName:        {authtypes.Minter, authtypes.Burner},
		apptypes.ModuleName:            {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		suppliertypes.ModuleName:       {authtypes.Minter, authtypes.Burner, authtypes.Staking},
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		macPerms,
		addrCodec,
		app.AccountAddressPrefix,
		authority.String(),
	)
	authModule := auth.NewAppModule(
		cdc,
		accountKeeper,
		authsims.RandomGenesisAccounts,
		nil, // subspace is nil because we don't test params (which is legacy anyway)
	)

	blockedAddresses := map[string]bool{
		accountKeeper.GetAuthority(): false,
	}
	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[banktypes.StoreKey]),
		accountKeeper,
		blockedAddresses,
		authority.String(),
		logger)

	stakingKeeper := stakingkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[stakingtypes.StoreKey]),
		accountKeeper,
		bankKeeper,
		authority.String(),
		addresscodec.NewBech32Codec(sdk.Bech32PrefixValAddr),
		addresscodec.NewBech32Codec(sdk.Bech32PrefixConsAddr),
	)
	stakingModule := staking.NewAppModule(cdc, stakingKeeper, accountKeeper, bankKeeper, nil)
	// slashingModule := slashing.NewAppModule(cdc, slashingKeeper, accountKeeper, bankKeeper, stakingKeeper, cdc.InterfaceRegistry(), cometInfoService)
	// evidenceModule := evidence.NewAppModule(cdc, *evidenceKeeper, cometInfoService)

	// mintKeeper := mintkeeper.NewKeeper(
	// 	cdc,
	// 	runtime.NewKVStoreService(storeKeys[minttypes.StoreKey]),
	// 	stakingKeeper, // stakingKeeper is nil because we don't test staking
	// 	accountKeeper,
	// 	bankKeeper,
	// 	authtypes.FeeCollectorName,
	// 	authority.String(),
	// )
	// mintModule := mint.NewAppModule(
	// 	cdc,
	// 	mintKeeper,
	// 	accountKeeper,
	// 	nil, // inflationKeeper is nil because we don't test inflation
	// 	nil, // subspace is nil because we don't test params (which is legacy anyway)
	// )

	sharedKeeper := sharedkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[apptypes.StoreKey]),
		logger,
		authority.String(),
	)
	sharedModule := shared.NewAppModule(
		cdc,
		sharedKeeper,
		accountKeeper,
		bankKeeper,
	)

	gatewayKeeper := gatewaykeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[apptypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
	)
	gatewayModule := gateway.NewAppModule(
		cdc,
		gatewayKeeper,
		accountKeeper,
		bankKeeper,
	)

	applicationKeeper := appkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[apptypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
		accountKeeper,
		gatewayKeeper,
		sharedKeeper,
	)
	applicationModule := application.NewAppModule(
		cdc,
		applicationKeeper,
		accountKeeper,
		bankKeeper,
	)

	proofKeeper := proofkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[prooftypes.StoreKey]),
		logger,
		authority.String(),
		nil, // sessionk
		applicationKeeper,
		accountKeeper,
		sharedKeeper,
	)
	proofModule := proof.NewAppModule(
		cdc,
		proofKeeper,
		accountKeeper,
	)

	tokenomicsKeeper := tokenomicskeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKeys[tokenomicstypes.StoreKey]),
		logger,
		authority.String(),
		bankKeeper,
		accountKeeper,
		applicationKeeper,
		proofKeeper,
	)
	tokenomicsModule := tokenomics.NewAppModule(
		cdc,
		tokenomicsKeeper,
		accountKeeper,
		bankKeeper,
	)

	msgRouter := baseapp.NewMsgServiceRouter()
	grpcRouter := baseapp.NewGRPCQueryRouter()

	// cometService := runtime.NewContextAwareCometInfoService()

	// stakingKeeper := stakingkeeper.NewKeeper(cdc, runtime.NewEnvironment(runtime.NewKVStoreService(keys[stakingtypes.StoreKey]), log.NewNopLogger(), runtime.EnvWithQueryRouterService(grpcRouter), runtime.EnvWithMsgRouterService(msgRouter)), accountKeeper, bankKeeper, authority.String(), addresscodec.NewBech32Codec(sdk.Bech32PrefixValAddr), addresscodec.NewBech32Codec(sdk.Bech32PrefixConsAddr), cometService)
	// require.NoError(t, stakingKeeper.Params.Set(newCtx, stakingtypes.DefaultParams()))

	// apptypes.RegisterMsgServer(integrationApp.MsgServiceRouter(), appkeeper.NewMsgServerImpl(applicationKeeper))

	// tokenomicstypes.RegisterQueryHandler(integrationApp.QueryRouter(), tokenomicsKeeper.NewQueryServerImpl(tokenomicsKeeper))

	// Auth Module - register query & message server

	// Mint Module - register query & message server

	// minttypes.RegisterQueryServer(integrationApp.QueryHelper(), mintkeeper.NewQueryServerImpl(mintKeeper))

	// create the application and register all the modules from the previous step

	modules := map[string]appmodule.AppModule{
		tokenomicstypes.ModuleName: tokenomicsModule,
		// banktypes.ModuleName:       bankModule,
		sharedtypes.ModuleName:  sharedModule,
		gatewaytypes.ModuleName: gatewayModule,
		// sessiontypes.ModuleName:    sessionModule,
		apptypes.ModuleName: applicationModule,
		// suppliertypes.ModuleName:   supplierModule,
		prooftypes.ModuleName: proofModule,
		authtypes.ModuleName:  authModule,
		// minttypes.ModuleName:    mintModule,
		stakingtypes.ModuleName: stakingModule,
	}

	integrationApp := integration.NewIntegrationApp(
		t,
		newCtx,
		logger,
		storeKeys,
		cdc,
		modules,
		msgRouter,
		grpcRouter,
	)

	authtypes.RegisterMsgServer(msgRouter, authkeeper.NewMsgServerImpl(accountKeeper))
	// minttypes.RegisterMsgServer(msgRouter, mintkeeper.NewMsgServerImpl(mintKeeper))
	tokenomicstypes.RegisterMsgServer(msgRouter, tokenomicskeeper.NewMsgServerImpl(tokenomicsKeeper))

	// minttypes.RegisterQueryServer(integrationApp.QueryHelper(), mintkeeper.NewQueryServerImpl(mintKeeper))

	params := tokenomicstypes.DefaultParams()
	params.ComputeUnitsToTokensMultiplier = 42

	req := &tokenomicstypes.MsgUpdateParam{
		Authority: authority.String(),
		Name:      "compute_units_to_tokens_multiplier",
		AsType:    &tokenomicstypes.MsgUpdateParam_AsInt64{AsInt64: 10},
	}

	result, err := integrationApp.RunMsg(
		req,
		integration.WithAutomaticFinalizeBlock(),
		integration.WithAutomaticCommit(),
	)
	require.NoError(t, err)

	// in this example the result is an empty response, a nil check is enough
	// in other cases, it is recommended to check the result value.
	require.NotNil(t, result, "unexpected nil result")

	// we now check the result
	resp := tokenomicstypes.MsgUpdateParamResponse{}
	err = cdc.Unmarshal(result.Value, &resp)
	require.NoError(t, err)

	sdkCtx := sdk.UnwrapSDKContext(integrationApp.Context())

	// we should also check the state of the application
	gotParams := tokenomicsKeeper.GetParams(sdkCtx)
	// require.NoError(t, err)

	// require.True(t, cmp.Equal(got, params), "expected mint params to be %v, got %v", params, got)
	fmt.Println(gotParams.ComputeUnitsToTokensMultiplier) // Output: 10000

}

package network

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	tmdb "github.com/cometbft/cometbft-db"
	tmrand "github.com/cometbft/cometbft/libs/rand"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	pruningtypes "github.com/cosmos/cosmos-sdk/store/pruning/types"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"pocket/app"
	"pocket/testutil/nullify"
	"pocket/testutil/sample"
	app_types "pocket/x/application/types"
	gateway_types "pocket/x/gateway/types"
	shared_types "pocket/x/shared/types"
	supplier_types "pocket/x/supplier/types"
)

type (
	Network = network.Network
	Config  = network.Config
)

// New creates instance with fully configured cosmos network.
// Accepts optional config, that will be used in place of the DefaultConfig() if provided.
func New(t *testing.T, configs ...Config) *Network {
	if len(configs) > 1 {
		panic("at most one config should be provided")
	}
	var cfg network.Config
	if len(configs) == 0 {
		cfg = DefaultConfig()
	} else {
		cfg = configs[0]
	}
	net, err := network.New(t, t.TempDir(), cfg)
	require.NoError(t, err)
	_, err = net.WaitForHeight(1)
	require.NoError(t, err)
	t.Cleanup(net.Cleanup)
	return net
}

// DefaultConfig will initialize config for the network with custom application,
// genesis and single validator. All other parameters are inherited from cosmos-sdk/testutil/network.DefaultConfig
func DefaultConfig() network.Config {
	var (
		encoding = app.MakeEncodingConfig()
		chainID  = "chain-" + tmrand.NewRand().Str(6)
	)
	return network.Config{
		Codec:             encoding.Marshaler,
		TxConfig:          encoding.TxConfig,
		LegacyAmino:       encoding.Amino,
		InterfaceRegistry: encoding.InterfaceRegistry,
		AccountRetriever:  authtypes.AccountRetriever{},
		AppConstructor: func(val network.ValidatorI) servertypes.Application {
			return app.New(
				val.GetCtx().Logger,
				tmdb.NewMemDB(),
				nil,
				true,
				map[int64]bool{},
				val.GetCtx().Config.RootDir,
				0,
				encoding,
				simtestutil.EmptyAppOptions{},
				baseapp.SetPruning(pruningtypes.NewPruningOptionsFromString(val.GetAppConfig().Pruning)),
				baseapp.SetMinGasPrices(val.GetAppConfig().MinGasPrices),
				baseapp.SetChainID(chainID),
			)
		},
		GenesisState:    app.ModuleBasics.DefaultGenesis(encoding.Marshaler),
		TimeoutCommit:   2 * time.Second,
		ChainID:         chainID,
		NumValidators:   1,
		BondDenom:       sdk.DefaultBondDenom,
		MinGasPrices:    fmt.Sprintf("0.000006%s", sdk.DefaultBondDenom),
		AccountTokens:   sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction),
		StakingTokens:   sdk.TokensFromConsensusPower(500, sdk.DefaultPowerReduction),
		BondedTokens:    sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction),
		PruningStrategy: pruningtypes.PruningOptionNothing,
		CleanupDir:      true,
		SigningAlgo:     string(hd.Secp256k1Type),
		KeyringOptions:  []keyring.Option{},
	}
}

// DefaultApplicationModuleGenesisState generates a GenesisState object with a given number of applications.
// It returns the populated GenesisState object.
func DefaultApplicationModuleGenesisState(t *testing.T, n int) *app_types.GenesisState {
	t.Helper()
	state := app_types.DefaultGenesis()
	for i := 0; i < n; i++ {
		stake := sdk.NewCoin("upokt", sdk.NewInt(int64(i+1)))
		application := app_types.Application{
			Address: sample.AccAddress(),
			Stake:   &stake,
		}
		nullify.Fill(&application)
		state.ApplicationList = append(state.ApplicationList, application)
	}
	return state
}

// DefaultGatewayModuleGenesisState generates a GenesisState object with a given number of gateways.
// It returns the populated GenesisState object.
func DefaultGatewayModuleGenesisState(t *testing.T, n int) *gateway_types.GenesisState {
	t.Helper()
	state := gateway_types.DefaultGenesis()
	for i := 0; i < n; i++ {
		stake := sdk.NewCoin("upokt", sdk.NewInt(int64(i)))
		gateway := gateway_types.Gateway{
			Address: strconv.Itoa(i),
			Stake:   &stake,
		}
		nullify.Fill(&gateway)
		state.GatewayList = append(state.GatewayList, gateway)
	}
	return state
}

// DefaultSupplierModuleGenesisState generates a GenesisState object with a given number of suppliers.
// It returns the populated GenesisState object.
func DefaultSupplierModuleGenesisState(t *testing.T, n int) *supplier_types.GenesisState {
	t.Helper()
	state := supplier_types.DefaultGenesis()
	for i := 0; i < n; i++ {
		stake := sdk.NewCoin("upokt", sdk.NewInt(int64(i)))
		gateway := shared_types.Supplier{
			Address: strconv.Itoa(i),
			Stake:   &stake,
		}
		nullify.Fill(&gateway)
		state.SupplierList = append(state.SupplierList, gateway)
	}
	return state
}

// Initialize an Account by sending it some funds from the validator in the network to the address provided
func InitAccount(t *testing.T, net *Network, addr sdk.AccAddress) {
	t.Helper()
	val := net.Validators[0]
	ctx := val.ClientCtx
	args := []string{
		fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
	}
	amount := sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(200)))
	_, err := clitestutil.MsgSendExec(ctx, val.Address, addr, amount, args...)
	require.NoError(t, err)
}

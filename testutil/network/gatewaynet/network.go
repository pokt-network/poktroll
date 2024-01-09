package gatewaynet

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	testcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/network/basenet"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/x/application/client/cli"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ network.InMemoryNetwork = (*inMemoryNetworkWithGateways)(nil)

// inMemoryNetworkWithGateways is an implementation of the InMemoryNetwork interface.
type inMemoryNetworkWithGateways struct {
	//baseInMemoryNetwork basenet.BaseInMemoryNetwork
	basenet.BaseInMemoryNetwork
}

// DefaultInMemoryNetworkConfig returns the default in-memory network configuration.
// This configuration should sufficient populate on-chain objects to support reasonable
// coverage around most session-oriented scenarios.
func DefaultInMemoryNetworkConfig(t *testing.T) *network.InMemoryNetworkConfig {
	t.Helper()

	return &network.InMemoryNetworkConfig{
		NumGateways:             5,
		NumSuppliers:            2,
		AppSupplierPairingRatio: 1,
	}
}

// NewInMemoryNetworkWithGateways creates a new in-memory network with the given configuration.
func NewInMemoryNetworkWithGateways(t *testing.T, cfg *network.InMemoryNetworkConfig) *inMemoryNetworkWithGateways {
	t.Helper()

	return &inMemoryNetworkWithGateways{
		BaseInMemoryNetwork: *basenet.NewBaseInMemoryNetwork(
			t, cfg, testkeyring.NewPreGeneratedAccountIterator(),
		),
	}
}

// DelegateAppToGateway delegates the application by address to the gateway by address.
func (memnet *inMemoryNetworkWithGateways) DelegateAppToGateway(
	t *testing.T,
	appBech32 string,
	gatewayBech32 string,
) {
	t.Helper()

	args := []string{
		gatewayBech32,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, appBech32),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(memnet.GetNetwork(t).Config.BondDenom, math.NewInt(10))).String()),
	}
	responseRaw, err := testcli.ExecTestCLICmd(memnet.GetClientCtx(t), cli.CmdDelegateToGateway(), args)
	require.NoError(t, err)
	var resp sdk.TxResponse
	require.NoError(t, memnet.GetCosmosNetworkConfig(t).Codec.UnmarshalJSON(responseRaw.Bytes(), &resp))
	require.NotNil(t, resp)
	require.NotNil(t, resp.TxHash)
	require.Equal(t, uint32(0), resp.Code)
}

// UndelegateAppFromGateway delegates the application by address from the gateway by address.
func (memnet *inMemoryNetworkWithGateways) UndelegateAppFromGateway(
	t *testing.T,
	appBech32 string,
	gatewayBech32 string,
) {

	args := []string{
		gatewayBech32,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, appBech32),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(memnet.GetCosmosNetworkConfig(t).BondDenom, math.NewInt(10))).String()),
	}
	responseRaw, err := testcli.ExecTestCLICmd(memnet.GetClientCtx(t), cli.CmdUndelegateFromGateway(), args)
	require.NoError(t, err)
	var resp sdk.TxResponse
	require.NoError(t, memnet.GetCosmosNetworkConfig(t).Codec.UnmarshalJSON(responseRaw.Bytes(), &resp))
	require.NotNil(t, resp)
	require.NotNil(t, resp.TxHash)
	require.Equal(t, uint32(0), resp.Code)
}

// Start initializes the in-memory network and performs the following setup:
//   - populates a new in-memory keyring with a sufficient number of pre-generated accounts.
//   - configures the application module's genesis state using addresses corresponding
//     to config.NumApplications number of the same pre-generated accounts which were
//     added to the keyring.
//   - configures the supplier module's genesis state using addresses corresponding to
//     config.NumSuppliers number of the same pre-generated accounts which were added
//     to the keyring.
//   - creates the on-chain accounts in the accounts module which correspond to the
//     pre-generated accounts which were added to the keyring.
func (memnet *inMemoryNetworkWithGateways) Start(_ context.Context, t *testing.T) {
	t.Helper()

	// Application module genesis state fixture data generation is independent
	// of that of the supplier module.
	if memnet.Config.AppSupplierPairingRatio > 0 {
		panic("AppSupplierPairingRatio must be 0 for inMemoryNetworkWithGateways, use NumApplications instead")
	}

	memnet.InitializeDefaults(t)
	memnet.CreateKeyringAccounts(t)

	// Configure gateway and application module genesis states.
	memnet.configureGatewayModuleGenesisState(t)
	memnet.configureAppModuleGenesisState(t)

	memnet.Network = network.New(t, *memnet.GetCosmosNetworkConfig(t))

	memnet.FundOnChainAccounts(t)
}

// TODO_IN_THIS_COMMIT: fix comment...
// GatewayModuleGenesisStateWithAddresses generates a GenesisState object with
// a gateway list full of gateways with the given addresses.
func (memnet *inMemoryNetworkWithGateways) configureGatewayModuleGenesisState(t *testing.T) {
	t.Helper()

	gatewayGenesisState := gatewaytypes.DefaultGenesis()
	for gatewayIdx := 0; gatewayIdx < memnet.Config.NumGateways; gatewayIdx++ {
		stake := sdk.NewCoin("upokt", sdk.NewInt(int64(gatewayIdx)))
		preGeneratedAcct, ok := memnet.PreGeneratedAccountIterator.Next()
		require.Truef(t, ok, "pre-generated accounts iterator exhausted")
		require.Truef(t, ok, "pre-generated accounts iterator exhausted")

		gateway := gatewaytypes.Gateway{
			Address: preGeneratedAcct.Address.String(),
			Stake:   &stake,
		}

		// TODO_CONSIDERATION: Evaluate whether we need `nullify.Fill` or if we should enforce `(gogoproto.nullable) = false` everywhere
		// nullify.Fill(&gateway)
		gatewayGenesisState.GatewayList = append(gatewayGenesisState.GatewayList, gateway)
	}

	gatewayGenesisBuffer, err := memnet.GetCosmosNetworkConfig(t).Codec.MarshalJSON(gatewayGenesisState)
	require.NoError(t, err)

	// Add supplier module genesis supplierGenesisState to the network config.
	memnet.GetCosmosNetworkConfig(t).GenesisState[gatewaytypes.ModuleName] = gatewayGenesisBuffer
}

func (memnet *inMemoryNetworkWithGateways) configureAppModuleGenesisState(t *testing.T) {
	t.Helper()

	require.NotEmptyf(t, memnet.GetCosmosNetworkConfig(t), "cosmos config not initialized, call #Start() first")
	require.NotEmptyf(t, memnet.PreGeneratedAccountIterator, "pre-generated accounts not initialized, call #Start() first")

	var appGenesisState = apptypes.DefaultGenesis()
	for appIdx := 0; appIdx < memnet.Config.GetNumApplications(t); appIdx++ {
		preGeneratedAcct, ok := memnet.PreGeneratedAccountIterator.Next()
		require.Truef(t, ok, "pre-generated accounts iterator exhausted")

		application := apptypes.Application{
			Address: preGeneratedAcct.Address.String(),
			Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(10000)},
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
				{Service: &sharedtypes.Service{Id: fmt.Sprintf("svc%d", appIdx)}},
				// NB: applications are staked for a service which no supplier is staked for.
				{Service: &sharedtypes.Service{Id: fmt.Sprintf("nosvc%d", appIdx)}},
			},
		}
		appGenesisState.ApplicationList = append(appGenesisState.ApplicationList, application)
	}
	appGenesisBuffer, err := memnet.Config.CosmosCfg.Codec.MarshalJSON(appGenesisState)
	require.NoError(t, err)

	// Add supplier and application module genesis appGenesisState to the network memnetConfig.
	memnet.GetCosmosNetworkConfig(t).GenesisState[apptypes.ModuleName] = appGenesisBuffer
}

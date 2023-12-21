package basenet

import (
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	testcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const warnNoModuleGenesisFmt = "WARN: no %s module genesis state found, if this is unexpected, ensure that genesis is populated before creating on-chain accounts"

// CreateKeyringAccounts populates the Keyring associated with the in-memory
// network with memnet.numKeyringAccounts() number of pre-generated accounts.
func (memnet *BaseInMemoryCosmosNetwork) CreateKeyringAccounts(t *testing.T) {
	t.Helper()

	if memnet.Config.Keyring == nil {
		t.Log("Keyring not initialized, using new in-memory keyring")

		// Construct an in-memory keyring so that it can be populated and used prior
		// to network start.
		memnet.Config.Keyring = keyring.NewInMemory(memnet.Config.CosmosCfg.Codec)
	} else {
		t.Log("Keyring already initialized, using existing keyring")
	}

	// Create memnet.NumKeyringAccounts() accounts in the configured keyring.
	accts := testkeyring.CreatePreGeneratedKeyringAccounts(
		t, memnet.Config.Keyring, memnet.Config.GetNumKeyringAccounts(t),
	)

	// Assign the memnet's pre-generated accounts to be a new pre-generated
	// accounts iterator containing only the accounts which were also created
	// in the keyring.
	memnet.PreGeneratedAccounts = testkeyring.NewPreGeneratedAccountIterator(accts...)
}

func (memnet *BaseInMemoryCosmosNetwork) CreateOnChainAccounts(t *testing.T) {
	t.Helper()

	net := memnet.GetNetwork(t)
	require.NotEmptyf(t, net, "in-memory cosmos testutil network not initialized yet, call #Start() first")

	supplierGenesisState := network.GetGenesisState[*suppliertypes.GenesisState](t, suppliertypes.ModuleName, memnet)
	if supplierGenesisState == nil {
		t.Logf(warnNoModuleGenesisFmt, "supplier")
	} else {
		memnet.InitSupplierAccountsWithSequence(t, supplierGenesisState.SupplierList...)

	}

	appGenesisState := network.GetGenesisState[*apptypes.GenesisState](t, apptypes.ModuleName, memnet)
	if appGenesisState == nil {
		t.Logf(warnNoModuleGenesisFmt, "application")
	} else {
		memnet.InitAppAccountsWithSequence(t, appGenesisState.ApplicationList...)
	}

	gatewayGenesisState := network.GetGenesisState[*gatewaytypes.GenesisState](t, gatewaytypes.ModuleName, memnet)
	if gatewayGenesisState == nil {
		t.Logf(warnNoModuleGenesisFmt, "gateway")
	} else {
		memnet.InitGatewayAccountsWithSequence(t, gatewayGenesisState.GatewayList...)
	}

	// need to wait for the account to be initialized in the next block
	require.NoError(t, net.WaitForNextBlock())
}

// InitAccountWithSequence initializes an Account by sending it some funds from
// the validator in the network to the address provided
func (memnet *BaseInMemoryCosmosNetwork) InitAccountWithSequence(
	t *testing.T,
	addr types.AccAddress,
) {
	t.Helper()

	signerAccountNumber := 0
	// TODO_IN_THIS_COMMIT: comment.. must use validator ctx because its keyring contains the validator key.
	clientCtx := memnet.Network.Validators[0].ClientCtx
	//clientCtx := memnet.GetClientCtx(t)
	net := memnet.GetNetwork(t)
	val := net.Validators[0]

	args := []string{
		fmt.Sprintf("--%s=true", flags.FlagOffline),
		fmt.Sprintf("--%s=%d", flags.FlagAccountNumber, signerAccountNumber),
		fmt.Sprintf("--%s=%d", flags.FlagSequence, memnet.NextAccountSequenceNumber(t)),

		fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, types.NewCoins(types.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}
	amount := types.NewCoins(types.NewCoin("stake", math.NewInt(200)))
	responseRaw, err := testcli.MsgSendExec(clientCtx, val.Address, addr, amount, args...)
	require.NoError(t, err)
	var responseJSON map[string]interface{}
	err = json.Unmarshal(responseRaw.Bytes(), &responseJSON)
	require.NoError(t, err)
	require.Equal(t, float64(0), responseJSON["code"], "code is not 0 in the response: %v", responseJSON)
}

func (memnet *BaseInMemoryCosmosNetwork) InitSupplierAccountsWithSequence(
	t *testing.T,
	supplierList ...sharedtypes.Supplier,
) {
	t.Helper()

	net := memnet.GetNetwork(t)
	require.NotNil(t, net, "in-memory cosmos testutil network not initialized yet, call #Start() first")

	for _, supplier := range supplierList {
		supplierAddr, err := types.AccAddressFromBech32(supplier.GetAddress())
		require.NoError(t, err)
		memnet.InitAccountWithSequence(t, supplierAddr)
	}
}

func (memnet *BaseInMemoryCosmosNetwork) InitAppAccountsWithSequence(
	t *testing.T,
	appList ...apptypes.Application,
) {
	t.Helper()

	for _, application := range appList {
		appAddr, err := types.AccAddressFromBech32(application.GetAddress())
		require.NoError(t, err)
		memnet.InitAccountWithSequence(t, appAddr)
	}
}

func (memnet *BaseInMemoryCosmosNetwork) InitGatewayAccountsWithSequence(
	t *testing.T,
	gatewayList ...gatewaytypes.Gateway,
) {
	t.Helper()

	for _, gateway := range gatewayList {
		gatewayAddr, err := types.AccAddressFromBech32(gateway.GetAddress())
		require.NoError(t, err)
		memnet.InitAccountWithSequence(t, gatewayAddr)
	}
}

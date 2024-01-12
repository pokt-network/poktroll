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
	sdk "github.com/cosmos/cosmos-sdk/types"
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
func (memnet *BaseInMemoryNetwork) CreateKeyringAccounts(t *testing.T) {
	t.Helper()

	// Keyring MAY be provided setting InMemoryNetworkConfig#Keyring.
	if memnet.Config.Keyring == nil {
		t.Log("Keyring not initialized, using new in-memory keyring")

		// Construct an in-memory keyring so that it can be populated and used prior
		// to network start.
		memnet.Config.Keyring = keyring.NewInMemory(memnet.Config.CosmosCfg.Codec)
	} else {
		t.Log("Keyring already initialized, using existing keyring")
	}

	// Create memnet.NumKeyringAccounts() number of accounts in the configured keyring.
	accts := testkeyring.CreatePreGeneratedKeyringAccounts(
		t, memnet.Config.Keyring, memnet.Config.GetNumKeyringAccounts(t),
	)

	// Assign the memnet's pre-generated accounts to a new pre-generated
	// accounts iterator containing only the accounts which were also created
	// in the keyring.
	memnet.PreGeneratedAccountIterator = testkeyring.NewPreGeneratedAccountIterator(accts...)
}

// FundOnChainAccounts creates on-chain accounts (i.e. auth module) for the sum of
// the configured number of suppliers, applications, and gateways.
func (memnet *BaseInMemoryNetwork) FundOnChainAccounts(t *testing.T) {
	t.Helper()

	// NB: while it may initially seem like the memnet#Fund<actor>Accounts() methods
	// can be refactored into a generic function, this is not possible so long as the genesis
	// state lists are passed directly & remain a slice of concrete types (as opposed to pointers).
	// Under these conditions, a generic function would not be able to unmarshal the genesis state
	// stored in the in-memory network because it is unmarshalling uses reflection, and it is not
	// possible to reflect over a nil generic type value.

	// Retrieve the supplier module's genesis state from cosmos-sdk in-memory network.
	supplierGenesisState := network.GetGenesisState[*suppliertypes.GenesisState](t, suppliertypes.ModuleName, memnet)
	if supplierGenesisState == nil {
		t.Logf(warnNoModuleGenesisFmt, "supplier")
	} else {
		// Initialize on-chain accounts for genesis suppliers.
		memnet.FundSupplierAccounts(t, supplierGenesisState.SupplierList...)

	}

	// Retrieve the application module's genesis state from cosmos-sdk in-memory network.
	appGenesisState := network.GetGenesisState[*apptypes.GenesisState](t, apptypes.ModuleName, memnet)
	if appGenesisState == nil {
		t.Logf(warnNoModuleGenesisFmt, "application")
	} else {
		// Initialize on-chain accounts for genesis applications.
		memnet.FundAppAccounts(t, appGenesisState.ApplicationList...)
	}

	// Retrieve the gateway module's genesis state from cosmos-sdk in-memory network.
	gatewayGenesisState := network.GetGenesisState[*gatewaytypes.GenesisState](t, gatewaytypes.ModuleName, memnet)
	if gatewayGenesisState == nil {
		t.Logf(warnNoModuleGenesisFmt, "gateway")
	} else {
		// Initialize on-chain accounts for genesis gateways.
		memnet.FundGatewayAccounts(t, gatewayGenesisState.GatewayList...)
	}

	// need to wait for the account to be initialized in the next block
	require.NoError(t, memnet.GetNetwork(t).WaitForNextBlock())
}

// FundAddress initializes an Account address and sequence number with the auth module
// by sending some tokens from the in-memory network  validator, to the address provided.
// This is a necessary prerequesite in order for the account with the given address
// to be able to submit valid transactions (i.e. pay tx fees).
// NOTE: It DOES NOT associate a public key with the account. This will happen when a tx
// which is signed by the account is broadcast to the network for the first time.
func (memnet *BaseInMemoryNetwork) FundAddress(
	t *testing.T,
	addr types.AccAddress,
) {
	t.Helper()

	signerAccountNumber := 0
	clientCtx := memnet.GetClientCtx(t)
	net := memnet.GetNetwork(t)
	val := net.Validators[0]

	args := []string{
		fmt.Sprintf("--%s=true", flags.FlagOffline),
		fmt.Sprintf("--%s=%d", flags.FlagAccountNumber, signerAccountNumber),
		fmt.Sprintf("--%s=%d", flags.FlagSequence, memnet.NextValidatorTxSequenceNumber(t)),

		fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, memnet.NewBondDenomCoins(t, 10).String()),
	}
	amount := memnet.NewBondDenomCoins(t, 200)
	responseRaw, err := testcli.MsgSendExec(clientCtx, val.Address, addr, amount, args...)
	require.NoError(t, err)
	var responseJSON map[string]interface{}
	err = json.Unmarshal(responseRaw.Bytes(), &responseJSON)
	require.NoError(t, err)
	require.Equal(t, float64(0), responseJSON["code"], "code is not 0 in the response: %v", responseJSON)
}

// FundSupplierAccounts initializes account addresses and sequence numbers for,
// each supplier in the given list, with the auth module by sending them some tokens
// from the in-memory network validator, to the address provided. This is a necessary
// prerequisite in order for the account with the given address to be able to submit
// valid transactions (i.e. pay tx fees).
// NOTE: It DOES NOT associate a public key with the account. This will happen when a tx
// which is signed by the account is broadcast to the network for the first time.
func (memnet *BaseInMemoryNetwork) FundSupplierAccounts(
	t *testing.T,
	supplierList ...sharedtypes.Supplier,
) {
	t.Helper()

	net := memnet.GetNetwork(t)
	require.NotNil(t, net, "in-memory cosmos testutil network not initialized yet, call #Start() first")

	for _, supplier := range supplierList {
		supplierAddr, err := types.AccAddressFromBech32(supplier.GetAddress())
		require.NoError(t, err)
		memnet.FundAddress(t, supplierAddr)
	}
}

// FundAppAccounts initializes account addresses and sequence numbers for, each
// application in the given list, with the auth module by sending them some tokens
// from the in-memory network validator, to the address provided. This is a necessary
// prerequisite in order for the account with the given address to be able to submit
// valid transactions (i.e. pay tx fees).
// NOTE: It DOES NOT associate a public key with the account. This will happen when a tx
// which is signed by the account is broadcast to the network for the first time.
func (memnet *BaseInMemoryNetwork) FundAppAccounts(
	t *testing.T,
	appList ...apptypes.Application,
) {
	t.Helper()

	for _, application := range appList {
		appAddr, err := types.AccAddressFromBech32(application.GetAddress())
		require.NoError(t, err)
		memnet.FundAddress(t, appAddr)
	}
}

// FundGatewayAccounts initializes account addresses and sequence numbers for, each
// gateway in the given list, with the auth module by sending them some tokens
// from the in-memory network validator, to the address provided. This is a necessary
// prerequisite in order for the account with the given address to be able to submit
// valid transactions (i.e. pay tx fees).
// NOTE: It DOES NOT associate a public key with the account. This will happen when a tx
// which is signed by the account is broadcast to the network for the first time.
func (memnet *BaseInMemoryNetwork) FundGatewayAccounts(
	t *testing.T,
	gatewayList ...gatewaytypes.Gateway,
) {
	t.Helper()

	for _, gateway := range gatewayList {
		gatewayAddr, err := types.AccAddressFromBech32(gateway.GetAddress())
		require.NoError(t, err)
		memnet.FundAddress(t, gatewayAddr)
	}
}

// CreateNewOnChainAccount uses the pre-generated account iterator associated
// with the in-memory network (which is also used to populate genesis and the
// in-memory network keyring). It initializes the address on-chain by sending
// it some tokens and also creates a corresponding keypair in the in-memory
// network's keyring. It returns the pre-generated account which was used.
func (memnet *BaseInMemoryNetwork) CreateNewOnChainAccount(t *testing.T) *testkeyring.PreGeneratedAccount {
	t.Helper()

	// Get the next available pre-generated account.
	preGeneratedAcct, ok := testkeyring.PreGeneratedAccounts().Next()
	require.Truef(t, ok, "no pre-generated accounts available")

	// Create an account in the auth module with the address of the pre-generated account.
	memnet.FundAddress(t, preGeneratedAcct.Address)

	// Create an entry in the keyring using the pre-generated account's mnemonic.
	testkeyring.CreatePreGeneratedKeyringAccounts(t, memnet.GetClientCtx(t).Keyring, 1)

	return preGeneratedAcct
}

// NewBondDenomCoins returns a Coins object containing the given number of coins
// in terms of the network's configured bond denomination.
func (memnet *BaseInMemoryNetwork) NewBondDenomCoins(t *testing.T, numCoins int64) sdk.Coins {
	t.Helper()

	return sdk.NewCoins(sdk.NewCoin(memnet.GetNetwork(t).Config.BondDenom, math.NewInt(numCoins)))
}

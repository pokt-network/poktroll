package sessionnet

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/x/shared/types"
	types2 "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	// TODO_IN_THIS_COMMIT: reconsider usage...
	testServiceId = "svc0"
)

var testProofPath = []byte("test_proof_path")

// TODO_IN_THIS_COMMIT: move
type inMemoryNetworkWithSessions struct {
	config               network.InMemoryNetworkConfig
	preGeneratedAccounts *testkeyring.PreGeneratedAccountIterator
	network              *network.Network
}

// TODO_IN_THIS_COMMIT: comment...
// TODO_IN_THIS_COMMIT: return interface type.
func DefaultNetworkWithSessions(t *testing.T) *inMemoryNetworkWithSessions {
	t.Helper()

	return NewInMemoryNetworkWithSessions(
		&network.InMemoryNetworkConfig{
			NumSessions:         4,
			NumRelaysPerSession: 5,
			NumBlocksPerSession: 4,
			NumSuppliers:        2,
			NumApplications:     3,
		},
	)
}

func NewInMemoryNetworkWithSessions(cfg *network.InMemoryNetworkConfig) *inMemoryNetworkWithSessions {
	return &inMemoryNetworkWithSessions{
		config:               *cfg,
		preGeneratedAccounts: testkeyring.NewPreGeneratedAccountIterator(),
	}
}

func (memnet *inMemoryNetworkWithSessions) Start(t *testing.T) {
	t.Helper()

	if memnet.config.CosmosCfg == nil {
		t.Log("Cosmos config not initialized, using default config")

		// Initialize a network config.
		cfg := network.DefaultConfig()
		memnet.config.CosmosCfg = &cfg
	} else {
		t.Log("Cosmos config already initialized, using existing config")
	}

	memnet.createKeyringAccounts(t)

	// Configure supplier and application module genesis states.
	memnet.configureAppModuleGenesisState(t)
	memnet.configureSupplierModuleGenesisState(t)

	memnet.network = network.New(t, *memnet.config.CosmosCfg)

	memnet.createOnChainAccounts(t)
}

// TODO_IN_THIS_COMMIT: do we need this?
func (memnet *inMemoryNetworkWithSessions) GetNetwork(t *testing.T) *network.Network {
	t.Helper()

	require.NotEmptyf(t, memnet.network, "in-memory network not started yet, call inMemoryNetworkWithSessions#Start() first")

	return memnet.network
}

func (memnet *inMemoryNetworkWithSessions) GetClientCtx(t *testing.T) client.Context {
	t.Helper()

	require.NotEmptyf(t, memnet.network, "in-memory network not started yet, call inMemoryNetworkWithSessions#Start() first")

	// Only the first validator's sdkclient context is populated.
	// (see: https://pkg.go.dev/github.com/cosmos/cosmos-sdk/testutil/network#pkg-overview)
	ctx := memnet.network.Validators[0].ClientCtx

	// Overwrite the sdkclient context's Keyring with the in-memory one that contains
	// our pre-generated accounts.
	return ctx.WithKeyring(memnet.config.Keyring)
}

// networkWithSupplierObjects creates a new network with a given number of supplier objects.
// It returns the network and a slice of the created supplier objects.
func NetworkWithSupplierObjects(t *testing.T, n int) (*network.Network, []types.Supplier) {
	t.Helper()

	memnet := DefaultNetworkWithSessions(t)
	memnet.config.NumSuppliers = n
	memnet.Start(t)

	supplierGenesisState := GetGenesisState[*types2.GenesisState](t, types2.ModuleName, memnet)

	return memnet.GetNetwork(t), supplierGenesisState.SupplierList
}

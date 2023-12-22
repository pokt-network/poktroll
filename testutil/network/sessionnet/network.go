package sessionnet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/network/basenet"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
)

var _ network.InMemoryCosmosNetwork = (*inMemoryNetworkWithSessions)(nil)

// inMemoryNetworkWithSessions is an implementation of the InMemoryCosmosNetwork interface.
type inMemoryNetworkWithSessions struct {
	basenet.BaseInMemoryCosmosNetwork
}

// DefaultInMemoryNetworkConfig returns the default in-memory network configuration.
// This configuration should sufficient populate on-chain objects to support reasonable
// coverage around most session-oriented scenarios.
func DefaultInMemoryNetworkConfig(t *testing.T) *network.InMemoryNetworkConfig {
	t.Helper()

	return &network.InMemoryNetworkConfig{
		NumSessions:             4,
		NumRelaysPerSession:     5,
		NumBlocksPerSession:     5,
		NumSuppliers:            2,
		AppSupplierPairingRatio: 2,
	}
}

// DefaultNetworkWithSessions creates a new in-memory network using the default configuration.
func DefaultNetworkWithSessions(t *testing.T) *inMemoryNetworkWithSessions {
	t.Helper()

	return NewInMemoryNetworkWithSessions(t, DefaultInMemoryNetworkConfig(t))
}

// NewInMemoryNetworkWithSessions creates a new in-memory network with the given configuration.
func NewInMemoryNetworkWithSessions(t *testing.T, cfg *network.InMemoryNetworkConfig) *inMemoryNetworkWithSessions {
	t.Helper()

	return &inMemoryNetworkWithSessions{
		BaseInMemoryCosmosNetwork: basenet.BaseInMemoryCosmosNetwork{
			Config:               *cfg,
			PreGeneratedAccounts: testkeyring.NewPreGeneratedAccountIterator(),
		},
	}
}

// Start initializes the in-memory network and performs the following setup:
//   - populates a new in-memory keyring with a sufficient number of pre-generated accounts.
//   - configures the application module's genesis state using addresses corresponding
//     to #GetNumApplications() number of the same pre-generated accounts which were
//     added to the keyring.
//   - configures the supplier module's genesis state using addresses corresponding to
//     config.NumSuppliers number of the same pre-generated accounts which were added
//     to the keyring.
//   - creates the on-chain accounts in the accounts module which correspond to the
//     pre-generated accounts that were added to the keyring.
func (memnet *inMemoryNetworkWithSessions) Start(_ context.Context, t *testing.T) {
	t.Helper()

	// Application module genesis state fixture data is generated in terms of
	// AppToSupplierRatio, and NumApplications cannot encode the distribution
	// of the application/supplier pairings.
	if memnet.Config.NumApplications > 0 {
		panic("NumApplications must be 0 for inMemoryNetworkWithSession, use AppToSupplierRatio instead")
	}

	memnet.InitializeDefaults(t)
	memnet.CreateKeyringAccounts(t)

	// Configure supplier and application module genesis states.
	memnet.configureAppModuleGenesisState(t)
	memnet.configureSupplierModuleGenesisState(t)

	memnet.Network = network.New(t, *memnet.GetNetworkConfig(t))
	err := memnet.Network.WaitForNextBlock()
	require.NoError(t, err)

	memnet.CreateOnChainAccounts(t)
}

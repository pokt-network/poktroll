package basenet

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	sdknetwork "github.com/cosmos/cosmos-sdk/testutil/network"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
)

var _ network.InMemoryNetwork = (*BaseInMemoryNetwork)(nil)

// BaseInMemoryNetwork is an "abstract" (i.e. partial) implementation, intended
// to be embedded by other ("concrete") InMemoryNetwork implementations.
type BaseInMemoryNetwork struct {
	Config                      network.InMemoryNetworkConfig
	PreGeneratedAccountIterator *testkeyring.PreGeneratedAccountIterator
	CosmosNetwork               *sdknetwork.Network

	// lastValidatorSeqNumber stores the last (most recently generated) account sequence number.
	// NB: explicitly NOT using atomic.Int32 as it's usage doesn't compose well with anonymous
	// literal declarations.
	lastValidatorSeqNumber int32
}

// NewBaseInMemoryNetwork creates a new BaseInMemoryNetwork with the given
// configuration and pre-generated accounts. Intended to be used in constructor
// functions of structs that embed BaseInMemoryNetwork.
func NewBaseInMemoryNetwork(
	t *testing.T,
	cfg *network.InMemoryNetworkConfig,
	preGeneratedAccounts *testkeyring.PreGeneratedAccountIterator,
) *BaseInMemoryNetwork {
	t.Helper()

	return &BaseInMemoryNetwork{
		Config:                      *cfg,
		PreGeneratedAccountIterator: preGeneratedAccounts,

		// First functional account sequence number is 1. Starting at 0 so that
		// callers can always use NextValidatorTxSequenceNumber() (no boundary condition).
		lastValidatorSeqNumber: int32(0),
	}
}

// InitializeDefaults sets the underlying cosmos-sdk testutil network config to
// a reasonable default in case one was not provided with the InMemoryNetworkConfig.
func (memnet *BaseInMemoryNetwork) InitializeDefaults(t *testing.T) {
	if memnet.Config.CosmosCfg != nil {
		t.Log("Cosmos config already initialized, using existing config")
		return
	}
	
	t.Log("Cosmos config not initialized, using default config")
	// Initialize a network config.
	cfg := network.DefaultConfig()
	memnet.Config.CosmosCfg = &cfg
}

// GetClientCtx returns the underlying cosmos-sdk testutil network's client context.
func (memnet *BaseInMemoryNetwork) GetClientCtx(t *testing.T) client.Context {
	t.Helper()

	// Only the first validator's client context is populated.
	// (see: https://pkg.go.dev/github.com/cosmos/cosmos-sdk/testutil/network#pkg-overview)
	ctx := memnet.GetNetwork(t).Validators[0].ClientCtx

	// TODO_NEXT(@bryanchriswhite): Ensure validator key is always available.

	// Overwrite the client context's Keyring with the in-memory one that contains
	// our pre-generated accounts.
	return ctx.WithKeyring(memnet.Config.Keyring)
}

// GetConfig returns the InMemoryNetworkConfig which associated with a given
// InMemoryNetwork instance.
func (memnet *BaseInMemoryNetwork) GetConfig(t *testing.T) *network.InMemoryNetworkConfig {
	t.Helper()

	return &memnet.Config
}

// GetCosmosNetworkConfig returns the underlying cosmos-sdk testutil network config.
// It requires that the config has been set, failing the test if not.
func (memnet *BaseInMemoryNetwork) GetCosmosNetworkConfig(t *testing.T) *sdknetwork.Config {
	t.Helper()

	require.NotEmptyf(t, memnet.Config, "in-memory network config not set")
	return memnet.Config.CosmosCfg
}

// GetNetwork returns the underlying cosmos-sdk testutil network instance.
// It requires that the cosmos-sdk in-memory network has been set, failing the test if not.
func (memnet *BaseInMemoryNetwork) GetNetwork(t *testing.T) *sdknetwork.Network {
	t.Helper()

	require.NotEmptyf(t, memnet.CosmosNetwork, "in-memory cosmos network not set")
	return memnet.CosmosNetwork
}

// GetLastValidatorTxSequenceNumber returns the last (most recently generated) account sequence number.
// It is safe for concurrent use.
func (memnet *BaseInMemoryNetwork) GetLastValidatorTxSequenceNumber(t *testing.T) int {
	t.Helper()

	return int(atomic.LoadInt32(&memnet.lastValidatorSeqNumber))
}

// NextValidatorTxSequenceNumber increments the account sequence number and returns the new value.
// It is safe for concurrent use.
func (memnet *BaseInMemoryNetwork) NextValidatorTxSequenceNumber(t *testing.T) int {
	t.Helper()

	return int(atomic.AddInt32(&memnet.lastValidatorSeqNumber, 1))
}

// Start is a stub which is expected to be implemented by "concrete" InMemoryNetwork
// implementations. As BaseInMemoryNetwork is intended to be an "abstract" implementation,
// it is too general to define this behavior, leaving it up to embedders. As a result,
// this function panics if it is called.
func (memnet *BaseInMemoryNetwork) Start(_ context.Context, t *testing.T) {
	panic("must be implemented by struct embedders")
}

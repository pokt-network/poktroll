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

var _ network.InMemoryCosmosNetwork = (*BaseInMemoryCosmosNetwork)(nil)

// BaseInMemoryCosmosNetwork is an "abstract" (i.e. partial) implementation, intended
// to be embedded by other ("concrete") InMemoryCosmosNetwork implementations.
type BaseInMemoryCosmosNetwork struct {
	Config               network.InMemoryNetworkConfig
	PreGeneratedAccounts *testkeyring.PreGeneratedAccountIterator
	Network              *sdknetwork.Network

	// lastAccountSeqNumber stores the last (most recently generated) account sequence number.
	// NB: explicitly NOT using atomic.Int32 as it's usage doesn't compose well with anonymous
	// literal declarations.
	lastAccountSeqNumber int32
}

// NewBaseInMemoryCosmosNetwork creates a new BaseInMemoryNetwork with the given
// configuration and pre-generated accounts. Intended to be used in constructor
// functions of structs that embed BaseInMemoryCosmosNetwork.
func NewBaseInMemoryCosmosNetwork(
	t *testing.T,
	cfg *network.InMemoryNetworkConfig,
	preGeneratedAccounts *testkeyring.PreGeneratedAccountIterator,
) *BaseInMemoryCosmosNetwork {
	t.Helper()

	return &BaseInMemoryCosmosNetwork{
		Config:               *cfg,
		PreGeneratedAccounts: preGeneratedAccounts,

		// First functional account sequence number is 1. Starting at 0 so that
		// callers can always use NextAccountSequenceNumber() (no boundary condition).
		lastAccountSeqNumber: int32(0),
	}
}

// InitializeDefaults sets the underlying cosmos-sdk testutil network config to
// a reasonable default in case one was not provided with the InMemoryNetworkConfig.
func (memnet *BaseInMemoryCosmosNetwork) InitializeDefaults(t *testing.T) {
	if memnet.Config.CosmosCfg == nil {
		t.Log("Cosmos config not initialized, using default config")

		// Initialize a network config.
		cfg := network.DefaultConfig()
		memnet.Config.CosmosCfg = &cfg
	} else {
		t.Log("Cosmos config already initialized, using existing config")
	}
}

// GetClientCtx returns the underlying cosmos-sdk testutil network's client context.
func (memnet *BaseInMemoryCosmosNetwork) GetClientCtx(t *testing.T) client.Context {
	t.Helper()

	// Only the first validator's client context is populated.
	// (see: https://pkg.go.dev/github.com/cosmos/cosmos-sdk/testutil/network#pkg-overview)
	ctx := memnet.GetNetwork(t).Validators[0].ClientCtx

	// TODO_UPNEXT(@bryanchriswhite): Ensure validator key is always available.

	// Overwrite the client context's Keyring with the in-memory one that contains
	// our pre-generated accounts.
	return ctx.WithKeyring(memnet.Config.Keyring)
}

// GetNetworkConfig returns the underlying cosmos-sdk testutil network config.
// It requires that the config has been set, failing the test if not.
func (memnet *BaseInMemoryCosmosNetwork) GetNetworkConfig(t *testing.T) *sdknetwork.Config {
	t.Helper()

	require.NotEmptyf(t, memnet.Config, "in-memory network config not set")
	return memnet.Config.CosmosCfg
}

// GetNetwork returns the underlying cosmos-sdk testutil network instance.
// It requires that the cosmos-sdk in-memory network has been set, failing the test if not.
func (memnet *BaseInMemoryCosmosNetwork) GetNetwork(t *testing.T) *sdknetwork.Network {
	t.Helper()

	require.NotEmptyf(t, memnet.Network, "in-memory cosmos network not set")
	return memnet.Network
}

// GetLastAccountSequenceNumber returns the last (most recently generated) account sequence number.
// It is safe for concurrent use.
func (memnet *BaseInMemoryCosmosNetwork) GetLastAccountSequenceNumber(t *testing.T) int {
	t.Helper()

	return int(atomic.LoadInt32(&memnet.lastAccountSeqNumber))
}

// NextAccountSequenceNumber increments the account sequence number and returns the new value.
// It is safe for concurrent use.
func (memnet *BaseInMemoryCosmosNetwork) NextAccountSequenceNumber(t *testing.T) int {
	t.Helper()

	return int(atomic.AddInt32(&memnet.lastAccountSeqNumber, 1))
}

// Start is a stub which is expected to be implemented by embedders. It panics when called.
func (memnet *BaseInMemoryCosmosNetwork) Start(_ context.Context, t *testing.T) {
	panic("not implemented")
}

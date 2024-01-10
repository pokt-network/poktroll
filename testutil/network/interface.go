package network

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/testutil/network"
)

// InMemoryNetwork encapsulates the cosmos-sdk testutil network instance and the
// responsibility of initializing it, along with (optional) additional/ setup, in
// #Start(). It also provides access to additional cosmos-sdk testutil network
// internals via corresponding methods.
type InMemoryNetwork interface {
	// GetConfig returns the InMemoryNetworkConfig which associated with a given
	// InMemoryNetwork instance.
	GetConfig(*testing.T) *InMemoryNetworkConfig

	// GetClientCtx returns a cosmos-sdk client.Context associated with the
	// underlying cosmos-sdk testutil network instance.
	GetClientCtx(*testing.T) client.Context

	// GetCosmosNetworkConfig returns the underlying cosmos-sdk testutil network config.
	GetCosmosNetworkConfig(*testing.T) *network.Config

	// GetNetwork returns the underlying cosmos-sdk testutil network instance.
	GetNetwork(*testing.T) *network.Network

	// Start initializes the in-memory network, performing any setup
	// (e.g. preparing on-chain state) for the test scenarios which
	// will be exercised afterward.
	Start(context.Context, *testing.T)
}

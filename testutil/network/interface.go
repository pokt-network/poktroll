package network

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/testutil/network"
)

// TODO_IN_THIS_COMMIT: move implementation to own pkg.
type InMemoryCosmosNetwork interface {
	GetClientCtx(*testing.T) client.Context
	GetNetwork(*testing.T) *network.Network
}

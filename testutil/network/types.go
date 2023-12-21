package network

import (
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/testutil/network"
)

// TODO_IN_THIS_COMMIT: godocs comment...
var TestProofPath = []byte("test_proof_path")

// TODO_IN_THIS_COMMIT: godocs comments...
type InMemoryNetworkConfig struct {
	NumSessions         int
	NumRelaysPerSession int
	NumBlocksPerSession int
	NumSuppliers        int
	NumGateways         int
	NumApplications     int
	// TODO_IN_THIS_COMMIT: comment ... w/ the **same serviceId**... mutually exclusive w/ NumApplications
	AppSupplierPairingRatio int
	CosmosCfg               *network.Config
	Keyring                 keyring.Keyring
}

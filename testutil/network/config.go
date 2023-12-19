package network

import (
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/testutil/network"
)

type Config = network.Config

type InMemoryNetworkConfig struct {
	NumSessions         int
	NumRelaysPerSession int
	NumBlocksPerSession int
	NumSuppliers        int
	NumApplications     int
	NumDelegates        int
	CosmosCfg           *Config
	Keyring             keyring.Keyring
}

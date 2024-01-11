package network

import (
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/testutil/network"
)

// TestMerkleProofPath is intended to be used as a "path" in a merkle tree for
// which to generate a proof.
var TestMerkleProofPath = []byte("test_proof_merkle_path")

// InMemoryNetworkConfig is a **SHARED** config struct for use by InMemoryNetwork
// implementations to configure themselves, provide the necessary parameters to set-up
// code, and initialize the underlying cosmos-sdk testutil network.
//
// Examples of set-up operations include but are not limited to:
// - Creating accounts in the local keyring.
// - Creating genesis state for (a) module(s).
// - Executing on-chain transactions (i.e. on-chain, non-genesis state).
// - Governance parameter configuration
// - Etc...
type InMemoryNetworkConfig struct {
	// NumSessions is the number of sessions (with increasing start heights) for
	// which the network should generate claims and proofs.
	NumSessions int

	// NumRelaysPerSession is the number of nodes to be inserted into each claim's
	// session tree prior to generating the corresponding proof.
	NumRelaysPerSession int

	// NumSuppliers is the number of suppliers that should be created at genesis.
	NumSuppliers int

	// NumGateways is the number of gateways that should be created at genesis.
	NumGateways int

	// NumApplications is the number of applications that should be created at genesis.
	// Usage is mutually exclusive with AppSupplierPairingRatio. This is enforced by
	// InMemoryNetwork implementations.
	//
	// NOTE: Use #GetNumApplications() to compute the correct value, regardless of
	// which field is used; in-memory network implementations SHOULD NOT modify config
	// fields. #GetNumApplications() is intended to be the single source of truth.
	NumApplications int

	// AppSupplierPairingRatio is the number of applications, per supplier, that
	// share a serviceId (i.e. will be in the same session).
	// Usage is mutually exclusive with NumApplications. This is enforced by
	// InMemoryNetwork implementations.
	//
	// NOTE: Use #GetNumApplications() to compute the correct value, regardless of
	// which field is used; in-memory network implementations SHOULD NOT modify config
	// fields. #GetNumApplications() is intended to be the single source of truth.
	AppSupplierPairingRatio int

	// CosmosCfg is the configuration for the underlying cosmos-sdk testutil network.
	CosmosCfg *network.Config

	// Keyring is the keyring to be used by clients of the cosmos-sdk testutil network.
	// It is intended to be populated with a sufficient number of accounts for the
	// InMemoryNetwork implementation's use cases. BaseInMemoryNetwork
	// implements a #GetNumKeyringAccounts() for this purpose.
	// This keyring is associated with the cosmos client context returned from
	// BaseInMemoryNetwork#GetClientCtx().
	Keyring keyring.Keyring
}

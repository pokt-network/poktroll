package proof_test

import (
	"strconv"
	"testing"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/network"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/proof/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	testServiceId = "svc1"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// TODO_CONSIDERATION: perhaps this (and/or other similar helpers) can be refactored
// into something more generic and moved into a shared testutil package.
// TODO_TECHDEBT: refactor; this function has more than a single responsibility,
// which should be to configure and start the test network. The genesis state,
// accounts, and claims set up logic can probably be factored out and/or reduced.
func networkWithClaimObjects(
	t *testing.T,
	numSessions int,
	numSuppliers int,
	numApps int,
) (net *network.Network, claims []types.Claim, clientCtx cosmosclient.Context) {
	t.Helper()

	// Initialize a network config.
	cfg := network.DefaultConfig()

	// Construct an in-memory keyring so that it can be populated and used prior
	// to network start.
	kr := keyring.NewInMemory(cfg.Codec)
	// Populate the in-memory keyring with as many pre-generated accounts as
	// we expect to need for the test (i.e. numApps + numSuppliers).
	testkeyring.CreatePreGeneratedKeyringAccounts(t, kr, numSuppliers+numApps)

	// Use the pre-generated accounts iterator to populate the supplier and
	// application accounts and addresses lists for use in genesis state construction.
	preGeneratedAccts := testkeyring.PreGeneratedAccounts().Clone()

	// Create a supplier for each session in numClaimsSessions and an app for each
	// claim in numClaimsPerSession.
	supplierOperatorAccts := make([]*testkeyring.PreGeneratedAccount, numSuppliers)
	supplierOperatorAddrs := make([]string, numSuppliers)
	for i := range supplierOperatorAccts {
		account, ok := preGeneratedAccts.Next()
		require.True(t, ok)

		supplierOperatorAccts[i] = account
		supplierOperatorAddrs[i] = account.Address.String()
	}
	appAccts := make([]*testkeyring.PreGeneratedAccount, numApps)
	appAddrs := make([]string, numApps)
	for i := range appAccts {
		account, ok := preGeneratedAccts.Next()
		require.True(t, ok)

		appAccts[i] = account
		appAddrs[i] = account.Address.String()
	}

	// Construct supplier and application module genesis states given the account addresses.
	supplierGenesisState := network.SupplierModuleGenesisStateWithAddresses(t, supplierOperatorAddrs)
	supplierGenesisBuffer, err := cfg.Codec.MarshalJSON(supplierGenesisState)
	require.NoError(t, err)

	appGenesisState := network.ApplicationModuleGenesisStateWithAddresses(t, appAddrs)
	appGenesisBuffer, err := cfg.Codec.MarshalJSON(appGenesisState)
	require.NoError(t, err)

	sharedParams := sharedtypes.DefaultParams()

	// Create numSessions * numApps * numSuppliers claims.
	for sessionIdx := 0; sessionIdx < numSessions; sessionIdx++ {
		for _, appAcct := range appAccts {
			for _, supplierOperatorAcct := range supplierOperatorAccts {
				claim := newTestClaim(
					t, &sharedParams,
					supplierOperatorAcct.Address.String(),
					testsession.GetSessionStartHeightWithDefaultParams(1),
					appAcct.Address.String(),
				)
				claims = append(claims, *claim)
			}
		}
	}

	// Add the claims to the proof module genesis state.
	proofGenesisState := network.ProofModuleGenesisStateWithClaims(t, claims)
	proofGenesisBuffer, err := cfg.Codec.MarshalJSON(proofGenesisState)
	require.NoError(t, err)

	// Add supplier and application module genesis states to the network config.
	cfg.GenesisState[suppliertypes.ModuleName] = supplierGenesisBuffer
	cfg.GenesisState[apptypes.ModuleName] = appGenesisBuffer
	cfg.GenesisState[prooftypes.ModuleName] = proofGenesisBuffer

	// Construct the network with the configuration.
	net = network.New(t, cfg)
	// Only the first validator's client context is populated.
	// (see: https://pkg.go.dev/github.com/cosmos/cosmos-sdk/testutil/network#pkg-overview)
	clientCtx = net.Validators[0].ClientCtx
	clientCtx = clientCtx.WithKeyring(kr)

	// Initialize all the accounts
	sequenceIndex := 1
	for _, supplierOperatorAcct := range supplierOperatorAccts {
		network.InitAccountWithSequence(t, net, supplierOperatorAcct.Address, sequenceIndex)
		sequenceIndex++
	}
	for _, appAcct := range appAccts {
		network.InitAccountWithSequence(t, net, appAcct.Address, sequenceIndex)
		sequenceIndex++
	}
	// need to wait for the account to be initialized in the next block
	require.NoError(t, net.WaitForNextBlock())

	return net, claims, clientCtx
}

// newTestClaim returns a new claim with the given supplier operator address, session start height,
// and application address. It uses mock byte slices for the root hash and block hash.
func newTestClaim(
	t *testing.T,
	sharedParams *sharedtypes.Params,
	supplierOperatorAddr string,
	sessionStartHeight int64,
	appAddr string,
) *types.Claim {
	t.Helper()

	// NB: These byte slices mock the root hash and block hash that would be
	// calculated and stored in the claim in a real scenario.
	rootHash := []byte("test_claim__mock_root_hash")
	blockHashBz := []byte("genesis_session__mock_block_hash")

	sessionId, _ := sessionkeeper.GetSessionId(
		sharedParams,
		appAddr,
		testServiceId,
		blockHashBz,
		0,
	)

	// TODO_TECHDEBT: Forward the actual claim in the response once the response is updated to return it.
	return &types.Claim{
		SupplierOperatorAddress: supplierOperatorAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      appAddr,
			ServiceId:               testServiceId,
			SessionId:               sessionId,
			SessionStartBlockHeight: sessionStartHeight,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(sessionStartHeight),
		},
		RootHash: rootHash,
	}
}

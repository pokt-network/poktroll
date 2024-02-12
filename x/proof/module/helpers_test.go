package proof_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	testcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	proof "github.com/pokt-network/poktroll/x/proof/module"
	"github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TODO_TECHDEBT: This should not be hardcoded once the num blocks per session is configurable.
const (
	numBlocksPerSession = 4
	testServiceId       = "svc1"
)

// Dummy variable to avoid unused import error.
var _ = strconv.IntSize

// init initializes the SDK configuration.
func init() {
	cmd.InitSDKConfig()
}

// TODO_CONSIDERATION: perhaps this (and/or other similar helpers) can be refactored
// into something more generic and moved into a shared testutil package.
// TODO_TECHDEBT: refactor; this function has more than a single responsibility,
// which should be to configure and start the test network. The genesis state,
// accounts, and claims set up logic can probably be factored out and/or reduced.
func networkWithClaimObjects(
	t *testing.T,
	sessionCount int,
	supplierCount int,
	appCount int,
) (net *network.Network, claims []types.Claim) {
	t.Helper()

	// Initialize a network config.
	cfg := network.DefaultConfig()

	// Construct an in-memory keyring so that it can be populated and used prior
	// to network start.
	kr := keyring.NewInMemory(cfg.Codec)
	// Populate the in-memmory keyring with as many pre-generated accounts as
	// we expect to need for the test (i.e. appCount + supplierCount).
	testkeyring.CreatePreGeneratedKeyringAccounts(t, kr, supplierCount+appCount)

	// Use the pre-generated accounts iterator to populate the supplier and
	// application accounts and addresses lists for use in genesis state construction.
	preGeneratedAccts := testkeyring.PreGeneratedAccounts().Clone()

	// Create a supplier for each session in numClaimsSessions and an app for each
	// claim in numClaimsPerSession.
	supplierAccts := make([]*testkeyring.PreGeneratedAccount, supplierCount)
	supplierAddrs := make([]string, supplierCount)
	for i := range supplierAccts {
		account, ok := preGeneratedAccts.Next()
		require.True(t, ok)
		supplierAccts[i] = account
		supplierAddrs[i] = account.Address.String()
	}
	appAccts := make([]*testkeyring.PreGeneratedAccount, appCount)
	appAddrs := make([]string, appCount)
	for i := range appAccts {
		account, ok := preGeneratedAccts.Next()
		require.True(t, ok)
		appAccts[i] = account
		appAddrs[i] = account.Address.String()
	}

	// Construct supplier and application module genesis states given the account addresses.
	supplierGenesisState := network.SupplierModuleGenesisStateWithAddresses(t, supplierAddrs)
	supplierGenesisBuffer, err := cfg.Codec.MarshalJSON(supplierGenesisState)
	require.NoError(t, err)
	appGenesisState := network.ApplicationModuleGenesisStateWithAddresses(t, appAddrs)
	appGenesisBuffer, err := cfg.Codec.MarshalJSON(appGenesisState)
	require.NoError(t, err)

	// Add supplier and application module genesis states to the network config.
	cfg.GenesisState[suppliertypes.ModuleName] = supplierGenesisBuffer
	cfg.GenesisState[apptypes.ModuleName] = appGenesisBuffer

	// Construct the network with the configuration.
	net = network.New(t, cfg)
	// Only the first validator's client context is populated.
	// (see: https://pkg.go.dev/github.com/cosmos/cosmos-sdk/testutil/network#pkg-overview)
	ctx := net.Validators[0].ClientCtx
	// Overwrite the client context's keyring with the in-memory one that contains
	// our pre-generated accounts.
	ctx = ctx.WithKeyring(kr)

	// Initialize all the accounts
	sequenceIndex := 1
	for _, supplierAcct := range supplierAccts {
		network.InitAccountWithSequence(t, net, supplierAcct.Address, sequenceIndex)
		sequenceIndex++
	}
	for _, appAcct := range appAccts {
		network.InitAccountWithSequence(t, net, appAcct.Address, sequenceIndex)
		sequenceIndex++
	}
	// need to wait for the account to be initialized in the next block
	require.NoError(t, net.WaitForNextBlock())

	// Create sessionCount * numClaimsPerSession claims for the supplier
	sessionEndHeight := int64(1)
	for sessionIdx := 0; sessionIdx < sessionCount; sessionIdx++ {
		sessionEndHeight += numBlocksPerSession
		for _, appAcct := range appAccts {
			for _, supplierAcct := range supplierAccts {
				claim := createClaim(
					t, net, ctx,
					supplierAcct.Address.String(),
					sessionEndHeight,
					appAcct.Address.String(),
				)
				claims = append(claims, *claim)
				// TODO_TECHDEBT(#196): Move this outside of the forloop so that the test iteration is faster
				require.NoError(t, net.WaitForNextBlock())
			}
		}
	}

	return net, claims
}

// encodeSessionHeader returns a base64 encoded string of a json
// serialized session header.
func encodeSessionHeader(
	t *testing.T,
	appAddr string,
	sessionId string,
	sessionStartHeight int64,
) string {
	t.Helper()

	argSessionHeader := &sessiontypes.SessionHeader{
		ApplicationAddress:      appAddr,
		SessionStartBlockHeight: sessionStartHeight,
		SessionId:               sessionId,
		SessionEndBlockHeight:   sessionStartHeight + numBlocksPerSession,
		Service:                 &sharedtypes.Service{Id: testServiceId},
	}
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	sessionHeaderBz := cdc.MustMarshalJSON(argSessionHeader)
	return base64.StdEncoding.EncodeToString(sessionHeaderBz)
}

// createClaim sends a tx using the test CLI to create an on-chain claim
func createClaim(
	t *testing.T,
	net *network.Network,
	ctx client.Context,
	supplierAddr string,
	sessionEndHeight int64,
	appAddress string,
) *types.Claim {
	t.Helper()

	rootHash := []byte("root_hash")
	sessionStartHeight := sessionEndHeight - numBlocksPerSession
	sessionId := getSessionId(t, net, appAddress, supplierAddr, sessionStartHeight)
	sessionHeaderEncoded := encodeSessionHeader(t, appAddress, sessionId, sessionStartHeight)
	rootHashEncoded := base64.StdEncoding.EncodeToString(rootHash)

	args := []string{
		sessionHeaderEncoded,
		rootHashEncoded,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, supplierAddr),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdktypes.NewCoins(sdktypes.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}

	responseRaw, err := testcli.ExecTestCLICmd(ctx, proof.CmdCreateClaim(), args)
	require.NoError(t, err)
	var responseJson map[string]interface{}
	err = json.Unmarshal(responseRaw.Bytes(), &responseJson)
	require.NoError(t, err)
	require.Equal(t, float64(0), responseJson["code"], "code is not 0 in the response: %v", responseJson)

	// TODO_TECHDEBT: Forward the actual claim in the response once the response is updated to return it.
	return &types.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      appAddress,
			Service:                 &sharedtypes.Service{Id: testServiceId},
			SessionId:               sessionId,
			SessionStartBlockHeight: sessionStartHeight,
			SessionEndBlockHeight:   sessionEndHeight,
		},
		RootHash: rootHash,
	}
}

// getSessionId sends a query using the test CLI to get a session for the inputs provided.
// It is assumed that the supplierAddr will be in that session based on the test design, but this
// is insured in this function before it's successfully returned.
func getSessionId(
	t *testing.T,
	net *network.Network,
	appAddr string,
	supplierAddr string,
	sessionStartHeight int64,
) string {
	t.Helper()
	ctx := context.TODO()

	sessionQueryClient := sessiontypes.NewQueryClient(net.Validators[0].ClientCtx)
	res, err := sessionQueryClient.GetSession(ctx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddr,
		Service:            &sharedtypes.Service{Id: testServiceId},
		BlockHeight:        sessionStartHeight,
	})
	require.NoError(t, err)

	var found bool
	for _, supplier := range res.GetSession().GetSuppliers() {
		if supplier.GetAddress() == supplierAddr {
			found = true
			break
		}
	}
	require.Truef(t, found, "supplier address %s not found in session", supplierAddr)

	return res.Session.SessionId
}

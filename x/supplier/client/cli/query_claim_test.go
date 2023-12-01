package cli_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	tmcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/client/cli"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// TODO_TECHDEBT: This should not be hardcoded once the num blocks per session is configurable
const numBlocksPerSession = 4

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
		Service:                 &sharedtypes.Service{Id: "svc1"},
	}
	cdc := codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
	sessionHeaderBz := cdc.MustMarshalJSON(argSessionHeader)
	return base64.StdEncoding.EncodeToString(sessionHeaderBz)
}

func createClaim(
	t *testing.T,
	net *network.Network,
	ctx client.Context,
	supplierAddr string,
	sessionEndHeight int64,
	appAddress string,
) *types.Claim {
	t.Helper()

	//appAddr := sample.AccAddress()
	rootHash := []byte("root_hash")
	sessionStartHeight := sessionEndHeight - numBlocksPerSession
	sessionId, err := getSessionId(t, net, appAddress, supplierAddr, sessionStartHeight)
	require.NoError(t, err)
	sessionHeaderEncoded := encodeSessionHeader(t, appAddress, sessionId, sessionStartHeight)
	rootHashEncoded := base64.StdEncoding.EncodeToString(rootHash)

	args := []string{
		sessionHeaderEncoded,
		rootHashEncoded,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, supplierAddr),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, sdkmath.NewInt(10))).String()),
	}

	responseRaw, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdCreateClaim(), args)
	require.NoError(t, err)
	var responseJson map[string]interface{}
	err = json.Unmarshal(responseRaw.Bytes(), &responseJson)
	require.NoError(t, err)
	require.Equal(t, float64(0), responseJson["code"], "code is not 0 in the response: %v", responseJson)

	return &types.Claim{
		SupplierAddress:       supplierAddr,
		SessionId:             sessionId,
		SessionEndBlockHeight: uint64(sessionEndHeight),
		RootHash:              rootHash,
	}
}

func getSessionId(
	t *testing.T,
	net *network.Network,
	appAddr string,
	supplierAddr string,
	sessionStartHeight int64,
) (string, error) {
	t.Helper()
	ctx := context.TODO()

	sessionQueryClient := sessiontypes.NewQueryClient(net.Validators[0].ClientCtx)
	res, err := sessionQueryClient.GetSession(ctx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddr,
		Service:            &sharedtypes.Service{Id: "svc1"}, // hardcoded for simplicity
		BlockHeight:        sessionStartHeight,
	})
	if err != nil {
		return "", err
	}

	var found bool
	for _, supplier := range res.GetSession().GetSuppliers() {
		if supplier.GetAddress() == supplierAddr {
			found = true
			break
		}
	}
	require.Truef(t, found, "supplier address %s not found in session", supplierAddr)

	return res.Session.SessionId, nil
}

// TODO_CONSIDERATION: perhaps this (and/or other similar helpers) can be refactored
// into something more generic and moved into a shared testutil package.
func networkWithClaimObjects(
	t *testing.T,
	numSessions int,
	numClaimsPerSession int,
) (net *network.Network, claims []types.Claim) {
	t.Helper()

	// Initialize a network config.
	cfg := network.DefaultConfig()

	// Construct an in-memory keyring so that it can be populated and used prior
	// to network start.
	kr := keyring.NewInMemory(cfg.Codec)
	// Populate the in-memmory keyring with as many pre-generated accounts as
	// we expect to need for the test.
	testkeyring.CreatePreGeneratedKeyringAccounts(t, kr, 20)

	// Use the pre-generated accounts iterator to populate the supplier and
	// application accounts and addresses lists for use in genesis state construction.
	preGeneratedAccts := testkeyring.PreGeneratedAccounts().Clone()

	// Create a supplier for each session in numSessions and an app for each
	// claim in numClaimsPerSession.
	supplierAccts := make([]*testkeyring.PreGeneratedAccount, numSessions)
	supplierAddrs := make([]string, numSessions)
	for i := range supplierAccts {
		account := preGeneratedAccts.MustNext()
		supplierAccts[i] = account
		supplierAddrs[i] = account.Address.String()
	}
	appAccts := make([]*testkeyring.PreGeneratedAccount, numClaimsPerSession)
	appAddrs := make([]string, numClaimsPerSession)
	for i := range appAccts {
		account := preGeneratedAccts.MustNext()
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
	cfg.GenesisState[types.ModuleName] = supplierGenesisBuffer
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

	// Create numSessions * numClaimsPerSession claims for the supplier
	sessionEndHeight := int64(1)
	for _, supplierAcct := range supplierAccts {
		sessionEndHeight += numBlocksPerSession
		for _, appAcct := range appAccts {
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

	return net, claims
}

func TestClaim_Show(t *testing.T) {
	numSessions := 1
	numClaimsPerSession := 2

	net, claims := networkWithClaimObjects(t, numSessions, numClaimsPerSession)

	ctx := net.Validators[0].ClientCtx
	common := []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	}
	tests := []struct {
		desc         string
		sessionId    string
		supplierAddr string

		args []string
		err  error
		obj  types.Claim
	}{
		{
			desc:         "claim found",
			sessionId:    claims[0].SessionId,
			supplierAddr: claims[0].SupplierAddress,

			args: common,
			obj:  claims[0],
		},
		{
			desc:         "claim not found (wrong session ID)",
			sessionId:    "wrong_session_id",
			supplierAddr: claims[0].SupplierAddress,

			args: common,
			err:  status.Error(codes.NotFound, "not found"),
		},
		{
			desc:         "claim not found (wrong supplier address)",
			sessionId:    claims[0].SessionId,
			supplierAddr: "wrong_supplier_address",

			args: common,
			err:  status.Error(codes.NotFound, "not found"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			args := []string{
				tc.sessionId,
				tc.supplierAddr,
			}
			args = append(args, tc.args...)
			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdShowClaim(), args)
			if tc.err != nil {
				stat, ok := status.FromError(tc.err)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), tc.err)
			} else {
				require.NoError(t, err)
				var resp types.QueryGetClaimResponse
				require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
				require.NotNil(t, resp.Claim)
				require.Equal(t,
					nullify.Fill(&tc.obj),
					nullify.Fill(&resp.Claim),
				)
			}
		})
	}
}

func TestClaim_List(t *testing.T) {
	numSessions := 2
	numClaimsPerSession := 5
	totalClaims := numSessions * numClaimsPerSession

	net, claims := networkWithClaimObjects(t, numSessions, numClaimsPerSession)

	ctx := net.Validators[0].ClientCtx
	prepareArgs := func(next []byte, offset, limit uint64, total bool) []string {
		args := []string{
			fmt.Sprintf("--%s=json", tmcli.OutputFlag),
		}
		if next == nil {
			args = append(args, fmt.Sprintf("--%s=%d", flags.FlagOffset, offset))
		} else {
			args = append(args, fmt.Sprintf("--%s=%s", flags.FlagPageKey, next))
		}
		args = append(args, fmt.Sprintf("--%s=%d", flags.FlagLimit, limit))
		if total {
			args = append(args, fmt.Sprintf("--%s", flags.FlagCountTotal))
		}
		return args
	}

	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < totalClaims; i += step {
			args := prepareArgs(nil, uint64(i), uint64(step), false)
			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
			require.NoError(t, err)

			var resp types.QueryAllClaimsResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

			require.LessOrEqual(t, len(resp.Claim), step)
			require.Subset(t,
				nullify.Fill(claims),
				nullify.Fill(resp.Claim),
			)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < totalClaims; i += step {
			args := prepareArgs(next, 0, uint64(step), false)
			out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
			require.NoError(t, err)

			var resp types.QueryAllClaimsResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

			require.LessOrEqual(t, len(resp.Claim), step)
			require.Subset(t,
				nullify.Fill(claims),
				nullify.Fill(resp.Claim),
			)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("ByAddress", func(t *testing.T) {
		supplierAddr := claims[0].SupplierAddress
		args := prepareArgs(nil, 0, uint64(totalClaims), true)
		args = append(args, fmt.Sprintf("--%s=%s", cli.FlagSupplierAddress, supplierAddr))

		expectedClaims := make([]types.Claim, 0)
		for _, claim := range claims {
			if claim.SupplierAddress == supplierAddr {
				expectedClaims = append(expectedClaims, claim)
			}
		}

		out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.ElementsMatch(t,
			nullify.Fill(expectedClaims),
			nullify.Fill(resp.Claim),
		)
		require.Equal(t, numClaimsPerSession, int(resp.Pagination.Total))
	})

	t.Run("BySession", func(t *testing.T) {
		sessionId := claims[0].SessionId
		args := prepareArgs(nil, 0, uint64(totalClaims), true)
		args = append(args, fmt.Sprintf("--%s=%s", cli.FlagSessionId, sessionId))

		expectedClaims := make([]types.Claim, 0)
		for _, claim := range claims {
			if claim.SessionId == sessionId {
				expectedClaims = append(expectedClaims, claim)
			}
		}

		out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.ElementsMatch(t,
			nullify.Fill(expectedClaims),
			nullify.Fill(resp.Claim),
		)
		require.Equal(t, 1, int(resp.Pagination.Total))
	})

	t.Run("ByHeight", func(t *testing.T) {
		sessionEndHeight := claims[0].SessionEndBlockHeight
		args := prepareArgs(nil, 0, uint64(totalClaims), true)
		args = append(args, fmt.Sprintf("--%s=%d", cli.FlagSessionEndHeight, sessionEndHeight))

		expectedClaims := make([]types.Claim, 0)
		for _, claim := range claims {
			if claim.SessionEndBlockHeight == sessionEndHeight {
				expectedClaims = append(expectedClaims, claim)
			}
		}

		out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.Equal(t, numClaimsPerSession, int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(expectedClaims),
			nullify.Fill(resp.Claim),
		)
	})

	t.Run("Total", func(t *testing.T) {
		args := prepareArgs(nil, 0, uint64(totalClaims), true)
		out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.Equal(t, totalClaims, int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(claims),
			nullify.Fill(resp.Claim),
		)
	})
}

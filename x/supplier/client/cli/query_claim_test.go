package cli_test

import (
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
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/client/cli"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// TODO_TECHDEBT: This should not be hardcoded once the num blocks per session is configurable
const numBlocksPerSession = 4

func encodeSessionHeader(t *testing.T, sessionId string, sessionEndHeight int64) string {
	t.Helper()

	argSessionHeader := &sessiontypes.SessionHeader{
		ApplicationAddress:      sample.AccAddress(),
		SessionStartBlockHeight: sessionEndHeight - numBlocksPerSession,
		SessionId:               sessionId,
		SessionEndBlockHeight:   sessionEndHeight,
		Service:                 &sharedtypes.Service{Id: "anvil"}, // hardcoded for simplicity
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
	sessionId string,
	sessionEndHeight int64,
) *types.Claim {
	t.Helper()

	rootHash := []byte("root_hash")
	sessionHeaderEncoded := encodeSessionHeader(t, sessionId, sessionEndHeight)
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

func networkWithClaimObjects(
	t *testing.T,
	numSessions int,
	numClaimsPerSession int,
) (net *network.Network, claims []types.Claim) {
	t.Helper()

	// Prepare the network
	cfg := network.DefaultConfig()
	net = network.New(t, cfg)
	ctx := net.Validators[0].ClientCtx

	// Prepare the keyring for the supplier account
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, numClaimsPerSession)
	ctx = ctx.WithKeyring(kr)

	// Initialize all the accounts
	for i, account := range accounts {
		signatureSequenceNumber := i + 1
		network.InitAccountWithSequence(t, net, account.Address, signatureSequenceNumber)
	}
	// need to wait for the account to be initialized in the next block
	require.NoError(t, net.WaitForNextBlock())

	addresses := make([]string, len(accounts))
	for i, account := range accounts {
		addresses[i] = account.Address.String()
	}

	// Create one supplier
	supplierGenesisState := network.SupplierModuleGenesisStateWithAccounts(t, addresses)
	buf, err := cfg.Codec.MarshalJSON(supplierGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf

	// Create numSessions * numClaimsPerSession claims for the supplier
	sessionEndHeight := int64(1)
	for sessionNum := 0; sessionNum < numSessions; sessionNum++ {
		sessionEndHeight += numBlocksPerSession
		sessionId := fmt.Sprintf("session_id%d", sessionNum)
		for claimNum := 0; claimNum < numClaimsPerSession; claimNum++ {
			supplierAddr := addresses[claimNum]
			claim := createClaim(t, net, ctx, supplierAddr, sessionId, sessionEndHeight)
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

		require.Equal(t, numSessions, int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(expectedClaims),
			nullify.Fill(resp.Claim),
		)
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

		require.Equal(t, numClaimsPerSession, int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(expectedClaims),
			nullify.Fill(resp.Claim),
		)
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

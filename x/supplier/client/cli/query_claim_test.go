package cli_test

import (
	"encoding/base64"
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

func encodeSessionHeader(t *testing.T, sessionId string, sessionEndHeight int64) string {
	t.Helper()

	argSessionHeader := &sessiontypes.SessionHeader{
		ApplicationAddress:      sample.AccAddress(),
		SessionStartBlockHeight: 1,
		SessionId:               sessionId,
		SessionEndBlockHeight:   sessionEndHeight,
		Service: &sharedtypes.Service{
			Id: "anvil",
		},
	}
	cdc := codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
	sessionHeaderBz := cdc.MustMarshalJSON(argSessionHeader)
	return base64.StdEncoding.EncodeToString(sessionHeaderBz)
}

func createClaim(t *testing.T, net *network.Network, ctx client.Context, supplierAddr string) *types.Claim {
	t.Helper()

	sessionEndHeight := int64(5)
	sessionId := "session_id"
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

	res, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdCreateClaim(), args)
	require.NoError(t, err)

	// TODO_IN_THIS_PR: Figure out why this still isn't working...
	fmt.Println("OLSH Claim created", res)
	return &types.Claim{
		SupplierAddress:       supplierAddr,
		SessionId:             sessionId,
		SessionEndBlockHeight: uint64(sessionEndHeight),
		RootHash:              rootHash,
	}
}

func networkWithClaimObjects(t *testing.T, n int) (net *network.Network, claims []types.Claim) {
	t.Helper()

	// Prepare the network
	cfg := network.DefaultConfig()
	net = network.New(t, cfg)
	ctx := net.Validators[0].ClientCtx

	// Prepare the keyring for the supplier account
	kr := ctx.Keyring
	accounts := testutil.CreateKeyringAccounts(t, kr, 1)
	supplierAccount := accounts[0]
	supplierAddress := supplierAccount.Address.String()

	// Update the context with the new keyring
	ctx = ctx.WithKeyring(kr)

	// Initialize the supplier account
	network.InitAccount(t, net, supplierAccount.Address)

	// Create one supplier
	supplierGenesisState := network.SupplierModuleGenesisStateWithAccount(t, supplierAddress)
	buf, err := cfg.Codec.MarshalJSON(supplierGenesisState)
	require.NoError(t, err)
	cfg.GenesisState[types.ModuleName] = buf

	// Create n claims for the supplier
	for i := 0; i < n; i++ {
		claim := createClaim(t, net, ctx, supplierAddress)
		claims = append(claims, *claim)
	}

	return net, claims
}

func TestClaim_Show(t *testing.T) {
	net, claims := networkWithClaimObjects(t, 2)

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
	net, claims := networkWithClaimObjects(t, 5)

	ctx := net.Validators[0].ClientCtx
	request := func(next []byte, offset, limit uint64, total bool) []string {
		args := []string{
			// fmt.Sprintf("--%s=json", tmcli.OutputFlag),
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
		for i := 0; i < len(claims); i += step {
			args := request(nil, uint64(i), uint64(step), false)
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
		for i := 0; i < len(claims); i += step {
			args := request(next, 0, uint64(step), false)
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
	t.Run("Total", func(t *testing.T) {
		args := request(nil, 0, uint64(len(claims)), true)
		out, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdListClaims(), args)
		require.NoError(t, err)

		var resp types.QueryAllClaimsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))

		require.Equal(t, len(claims), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(claims),
			nullify.Fill(resp.Claim),
		)
	})
}

// TODO_IN_THIS_PR: Add tests that query when querying with address/session/height filters

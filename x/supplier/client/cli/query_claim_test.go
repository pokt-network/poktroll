package cli_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	tmcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/nullify"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/client/cli"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func encodeSessionHeader(
	t *testing.T,
) string {
	t.Helper()
	argSessionHeader := &sessiontypes.SessionHeader{
		ApplicationAddress:      "pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4",
		SessionStartBlockHeight: 1,
		SessionId:               "session_id",
		SessionEndBlockHeight:   5,
		Service: &sharedtypes.Service{
			Id: "anvil",
		},
	}
	cdc := codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
	sessionHeaderBz := cdc.MustMarshalJSON(argSessionHeader)
	return hex.EncodeToString(sessionHeaderBz)
}

func createClaim(t *testing.T, ctx client.Context, supplierAddr string) *types.Claim {
	t.Helper()

	sessionHeaderEncoded := encodeSessionHeader(t)

	rootHash := []byte("root_hash")
	rootHashEncoded := hex.EncodeToString(rootHash)

	args := []string{
		sessionHeaderEncoded,
		rootHashEncoded,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, supplierAddr),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin("upokt", sdkmath.NewInt(10))).String()),
	}

	_, err := clitestutil.ExecTestCLICmd(ctx, cli.CmdCreateClaim(), args)
	require.NoError(t, err)

	return &types.Claim{
		SupplierAddress:       "pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4",
		SessionId:             "session_id",
		SessionEndBlockHeight: 5,
		RootHash:              rootHash,
	}
}

func networkWithClaimObjects(t *testing.T, n int) (net *network.Network, claims []types.Claim) {
	t.Helper()

	cfg := network.DefaultConfig()
	net = network.New(t, cfg)
	validator := net.Validators[0]
	ctx := validator.ClientCtx

	for i := 0; i < n; i++ {
		claim := createClaim(t, ctx, validator.Address.String())
		claims = append(claims, *claim)
	}

	return net, claims
}

func TestShowClaim(t *testing.T) {
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

func TestListClaim(t *testing.T) {
	net, claims := networkWithClaimObjects(t, 5)

	ctx := net.Validators[0].ClientCtx
	request := func(next []byte, offset, limit uint64, total bool) []string {
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
		require.NoError(t, err)
		require.Equal(t, len(claims), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(claims),
			nullify.Fill(resp.Claim),
		)
	})
}

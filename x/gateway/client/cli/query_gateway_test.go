package cli_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	tmcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client/flags"
	testcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/testutil/network"
	"github.com/pokt-network/poktroll/testutil/network/gatewaynet"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/gateway/client/cli"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestShowGateway(t *testing.T) {
	ctx := context.Background()
	memnet := gatewaynet.NewInMemoryNetworkWithGateways(
		t, &network.InMemoryNetworkConfig{
			NumGateways: 2,
		},
	)
	memnet.Start(ctx, t)

	clientCtx := memnet.GetClientCtx(t)
	codec := memnet.GetCosmosNetworkConfig(t).Codec
	gateways := network.GetGenesisState[*gatewaytypes.GenesisState](t, gatewaytypes.ModuleName, memnet).GatewayList

	common := []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	}
	tests := []struct {
		desc      string
		idAddress string

		args []string
		err  error
		obj  gatewaytypes.Gateway
	}{
		{
			desc:      "found",
			idAddress: gateways[0].Address,

			args: common,
			obj:  gateways[0],
		},
		{
			desc:      "not found",
			idAddress: strconv.Itoa(100000),

			args: common,
			err:  status.Error(codes.NotFound, "not found"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			args := []string{
				tc.idAddress,
			}
			args = append(args, tc.args...)
			out, err := testcli.ExecTestCLICmd(clientCtx, cli.CmdShowGateway(), args)
			if tc.err != nil {
				stat, ok := status.FromError(tc.err)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), tc.err)
			} else {
				require.NoError(t, err)
				var resp gatewaytypes.QueryGetGatewayResponse
				require.NoError(t, codec.UnmarshalJSON(out.Bytes(), &resp))
				require.NotNil(t, resp.Gateway)
				require.Equal(t,
					nullify.Fill(&tc.obj),
					nullify.Fill(&resp.Gateway),
				)
			}
		})
	}
}

func TestListGateway(t *testing.T) {
	ctx := context.Background()
	memnet := gatewaynet.NewInMemoryNetworkWithGateways(
		t, &network.InMemoryNetworkConfig{
			NumGateways: 5,
		},
	)
	memnet.Start(ctx, t)

	clientCtx := memnet.GetClientCtx(t)
	codec := memnet.GetCosmosNetworkConfig(t).Codec
	gateways := network.GetGenesisState[*gatewaytypes.GenesisState](t, gatewaytypes.ModuleName, memnet).GatewayList

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
		for i := 0; i < len(gateways); i += step {
			args := request(nil, uint64(i), uint64(step), false)
			out, err := testcli.ExecTestCLICmd(clientCtx, cli.CmdListGateway(), args)
			require.NoError(t, err)
			var resp gatewaytypes.QueryAllGatewayResponse
			require.NoError(t, codec.UnmarshalJSON(out.Bytes(), &resp))
			require.LessOrEqual(t, len(resp.Gateway), step)
			require.Subset(t,
				nullify.Fill(gateways),
				nullify.Fill(resp.Gateway),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(gateways); i += step {
			args := request(next, 0, uint64(step), false)
			out, err := testcli.ExecTestCLICmd(clientCtx, cli.CmdListGateway(), args)
			require.NoError(t, err)
			var resp gatewaytypes.QueryAllGatewayResponse
			require.NoError(t, codec.UnmarshalJSON(out.Bytes(), &resp))
			require.LessOrEqual(t, len(resp.Gateway), step)
			require.Subset(t,
				nullify.Fill(gateways),
				nullify.Fill(resp.Gateway),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		args := request(nil, 0, uint64(len(gateways)), true)
		out, err := testcli.ExecTestCLICmd(clientCtx, cli.CmdListGateway(), args)
		require.NoError(t, err)
		var resp gatewaytypes.QueryAllGatewayResponse
		require.NoError(t, codec.UnmarshalJSON(out.Bytes(), &resp))
		require.NoError(t, err)
		require.Equal(t, len(gateways), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(gateways),
			nullify.Fill(resp.Gateway),
		)
	})
}

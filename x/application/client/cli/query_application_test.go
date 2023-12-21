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
	"github.com/pokt-network/poktroll/testutil/network/sessionnet"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/application/client/cli"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

func TestShowApplication(t *testing.T) {
	ctx := context.Background()
	memnet := sessionnet.NewInMemoryNetworkWithSessions(
		t, &network.InMemoryNetworkConfig{
			NumSuppliers:            2,
			AppSupplierPairingRatio: 1,
		},
	)
	memnet.Start(ctx, t)

	appGenesisState := network.GetGenesisState[*apptypes.GenesisState](t, apptypes.ModuleName, memnet)
	applications := appGenesisState.ApplicationList

	net := memnet.GetNetwork(t)
	common := []string{
		fmt.Sprintf("--%s=json", tmcli.OutputFlag),
	}
	tests := []struct {
		desc      string
		idAddress string

		args []string
		err  error
		obj  apptypes.Application
	}{
		{
			desc:      "found",
			idAddress: applications[0].Address,

			args: common,
			obj:  applications[0],
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
			out, err := testcli.ExecTestCLICmd(memnet.GetClientCtx(t), cli.CmdShowApplication(), args)
			if tc.err != nil {
				stat, ok := status.FromError(tc.err)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), tc.err)
			} else {
				require.NoError(t, err)
				var resp apptypes.QueryGetApplicationResponse
				require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
				require.NotNil(t, resp.Application)
				require.Equal(t,
					nullify.Fill(&tc.obj),
					nullify.Fill(&resp.Application),
				)
			}
		})
	}
}

func TestListApplication(t *testing.T) {
	ctx := context.Background()
	memnet := sessionnet.NewInMemoryNetworkWithSessions(
		t, &network.InMemoryNetworkConfig{
			NumSuppliers:            5,
			AppSupplierPairingRatio: 1,
		},
	)
	memnet.Start(ctx, t)

	appGenesisState := network.GetGenesisState[*apptypes.GenesisState](t, apptypes.ModuleName, memnet)
	applications := appGenesisState.ApplicationList

	net := memnet.GetNetwork(t)
	clientCtx := memnet.GetClientCtx(t)

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
		for i := 0; i < len(applications); i += step {
			args := request(nil, uint64(i), uint64(step), false)
			out, err := testcli.ExecTestCLICmd(clientCtx, cli.CmdListApplication(), args)
			require.NoError(t, err)
			var resp apptypes.QueryAllApplicationResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
			require.LessOrEqual(t, len(resp.Application), step)
			require.Subset(t,
				nullify.Fill(applications),
				nullify.Fill(resp.Application),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(applications); i += step {
			args := request(next, 0, uint64(step), false)
			out, err := testcli.ExecTestCLICmd(clientCtx, cli.CmdListApplication(), args)
			require.NoError(t, err)
			var resp apptypes.QueryAllApplicationResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
			require.LessOrEqual(t, len(resp.Application), step)
			require.Subset(t,
				nullify.Fill(applications),
				nullify.Fill(resp.Application),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		args := request(nil, 0, uint64(len(applications)), true)
		out, err := testcli.ExecTestCLICmd(clientCtx, cli.CmdListApplication(), args)
		require.NoError(t, err)
		var resp apptypes.QueryAllApplicationResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
		require.NoError(t, err)
		require.Equal(t, len(applications), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(applications),
			nullify.Fill(resp.Application),
		)
	})
}

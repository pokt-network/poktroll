package application_test

import (
	"fmt"
	"strconv"
	"testing"

	cometcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/testutil/nullify"
	appmodule "github.com/pokt-network/poktroll/x/application/module"
)

func TestShowApplication(t *testing.T) {
	net, apps := networkWithApplicationObjects(t, 2)

	ctx := net.Validators[0].ClientCtx
	common := []string{
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	tests := []struct {
		desc      string
		idAddress string

		args        []string
		expectedErr error
		app         application.Application
	}{
		{
			desc:      "found",
			idAddress: apps[0].Address,

			args: common,
			app:  apps[0],
		},
		{
			desc:      "not found",
			idAddress: strconv.Itoa(100000),

			args:        common,
			expectedErr: status.Error(codes.NotFound, "not found"),
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			args := []string{
				test.idAddress,
			}
			args = append(args, test.args...)
			out, err := clitestutil.ExecTestCLICmd(ctx, appmodule.CmdShowApplication(), args)
			if test.expectedErr != nil {
				stat, ok := status.FromError(test.expectedErr)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), test.expectedErr)
			} else {
				require.NoError(t, err)
				var resp application.QueryGetApplicationResponse
				require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
				require.NotNil(t, resp.Application)
				require.Equal(t,
					nullify.Fill(&test.app),
					nullify.Fill(&resp.Application),
				)
			}
		})
	}
}

func TestListApplication(t *testing.T) {
	net, apps := networkWithApplicationObjects(t, 5)

	ctx := net.Validators[0].ClientCtx
	request := func(next []byte, offset, limit uint64, total bool) []string {
		args := []string{
			fmt.Sprintf("--%s=json", cometcli.OutputFlag),
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
		for i := 0; i < len(apps); i += step {
			args := request(nil, uint64(i), uint64(step), false)
			out, err := clitestutil.ExecTestCLICmd(ctx, appmodule.CmdListApplication(), args)
			require.NoError(t, err)
			var resp application.QueryAllApplicationsResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
			require.LessOrEqual(t, len(resp.Applications), step)
			require.Subset(t,
				nullify.Fill(apps),
				nullify.Fill(resp.Applications),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(apps); i += step {
			args := request(next, 0, uint64(step), false)
			out, err := clitestutil.ExecTestCLICmd(ctx, appmodule.CmdListApplication(), args)
			require.NoError(t, err)
			var resp application.QueryAllApplicationsResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
			require.LessOrEqual(t, len(resp.Applications), step)
			require.Subset(t,
				nullify.Fill(apps),
				nullify.Fill(resp.Applications),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		args := request(nil, 0, uint64(len(apps)), true)
		out, err := clitestutil.ExecTestCLICmd(ctx, appmodule.CmdListApplication(), args)
		require.NoError(t, err)
		var resp application.QueryAllApplicationsResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
		require.NoError(t, err)
		require.Equal(t, len(apps), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(apps),
			nullify.Fill(resp.Applications),
		)
	})
}

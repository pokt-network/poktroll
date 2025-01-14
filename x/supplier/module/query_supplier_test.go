package supplier_test

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

	"github.com/pokt-network/poktroll/testutil/nullify"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	supplier "github.com/pokt-network/poktroll/x/supplier/module"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func TestShowSupplier(t *testing.T) {
	net, suppliers := networkWithSupplierObjects(t, 2)

	ctx := net.Validators[0].ClientCtx
	common := []string{
		fmt.Sprintf("--%s=json", cometcli.OutputFlag),
	}
	tests := []struct {
		desc      string
		idAddress string

		args        []string
		expectedErr error
		supplier    sharedtypes.Supplier
	}{
		{
			desc:      "supplier found",
			idAddress: suppliers[0].OperatorAddress,

			args:     common,
			supplier: suppliers[0],
		},
		{
			desc:      "supplier not found",
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
			out, err := clitestutil.ExecTestCLICmd(ctx, supplier.CmdShowSupplier(), args)
			if test.expectedErr != nil {
				stat, ok := status.FromError(test.expectedErr)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), test.expectedErr)
			} else {
				require.NoError(t, err)
				var resp types.QueryGetSupplierResponse
				require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
				require.NotNil(t, resp.Supplier)
				require.Equal(t,
					nullify.Fill(&test.supplier),
					nullify.Fill(&resp.Supplier),
				)
			}
		})
	}
}

func TestListSuppliers(t *testing.T) {
	net, suppliers := networkWithSupplierObjects(t, 5)

	ctx := net.Validators[0].ClientCtx
	request := func(
		next []byte,
		offset,
		limit uint64,
		total bool,
		serviceId string,
	) []string {
		// Build the base args for the command
		args := []string{
			fmt.Sprintf("--%s=json", cometcli.OutputFlag),
			fmt.Sprintf("--%s=%d", flags.FlagLimit, limit),
		}

		// Add pagination flags if they're set
		if next == nil {
			args = append(args, fmt.Sprintf("--%s=%d", flags.FlagOffset, offset))
		} else {
			args = append(args, fmt.Sprintf("--%s=%s", flags.FlagPageKey, next))
		}

		// Add the total flag if it's set
		if total {
			args = append(args, fmt.Sprintf("--%s", flags.FlagCountTotal))
		}

		// Add the service ID if it's set
		if serviceId != "" {
			args = append(args, fmt.Sprintf("--%s=%s", supplier.FlagServiceId, serviceId))
		}

		return args
	}

	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(suppliers); i += step {
			args := request(nil, uint64(i), uint64(step), false, "")
			out, err := clitestutil.ExecTestCLICmd(ctx, supplier.CmdListSuppliers(), args)
			require.NoError(t, err)
			var resp types.QueryAllSuppliersResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
			require.LessOrEqual(t, len(resp.Supplier), step)
			require.Subset(t,
				nullify.Fill(suppliers),
				nullify.Fill(resp.Supplier),
			)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(suppliers); i += step {
			args := request(next, 0, uint64(step), false, "")
			out, err := clitestutil.ExecTestCLICmd(ctx, supplier.CmdListSuppliers(), args)
			require.NoError(t, err)
			var resp types.QueryAllSuppliersResponse
			require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
			require.LessOrEqual(t, len(resp.Supplier), step)
			require.Subset(t,
				nullify.Fill(suppliers),
				nullify.Fill(resp.Supplier),
			)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("Total", func(t *testing.T) {
		args := request(nil, 0, uint64(len(suppliers)), true, "")
		out, err := clitestutil.ExecTestCLICmd(ctx, supplier.CmdListSuppliers(), args)
		require.NoError(t, err)
		var resp types.QueryAllSuppliersResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
		require.NoError(t, err)
		require.Equal(t, len(suppliers), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(suppliers),
			nullify.Fill(resp.Supplier),
		)
	})

	t.Run("Filter By ServiceId", func(t *testing.T) {
		fmt.Println("OLSH", suppliers[0].Services)
		serviceId := suppliers[0].Services[0].ServiceId

		args := request(nil, 0, uint64(len(suppliers)), false, serviceId)
		_, err := clitestutil.ExecTestCLICmd(ctx, supplier.CmdListSuppliers(), args)
		require.NoError(t, err)

		// var resp types.QueryAllSuppliersResponse
		// require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
		// require.NoError(t, err)
		// require.Equal(t, 1, int(resp.Pagination.Total))
		// require.ElementsMatch(t,
		// 	nullify.Fill(suppliers),
		// 	nullify.Fill(resp.Supplier),
		// )
	})
}

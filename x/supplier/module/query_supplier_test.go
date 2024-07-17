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

	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/proto/types/supplier"
	"github.com/pokt-network/poktroll/testutil/nullify"
	suppliermodule "github.com/pokt-network/poktroll/x/supplier/module"
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
		supplier    shared.Supplier
	}{
		{
			desc:      "supplier found",
			idAddress: suppliers[0].Address,

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
			out, err := clitestutil.ExecTestCLICmd(ctx, suppliermodule.CmdShowSupplier(), args)
			if test.expectedErr != nil {
				stat, ok := status.FromError(test.expectedErr)
				require.True(t, ok)
				require.ErrorIs(t, stat.Err(), test.expectedErr)
			} else {
				require.NoError(t, err)
				var resp supplier.QueryGetSupplierResponse
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
		for i := 0; i < len(suppliers); i += step {
			args := request(nil, uint64(i), uint64(step), false)
			out, err := clitestutil.ExecTestCLICmd(ctx, suppliermodule.CmdListSuppliers(), args)
			require.NoError(t, err)
			var resp supplier.QueryAllSuppliersResponse
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
			args := request(next, 0, uint64(step), false)
			out, err := clitestutil.ExecTestCLICmd(ctx, suppliermodule.CmdListSuppliers(), args)
			require.NoError(t, err)
			var resp supplier.QueryAllSuppliersResponse
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
		args := request(nil, 0, uint64(len(suppliers)), true)
		out, err := clitestutil.ExecTestCLICmd(ctx, suppliermodule.CmdListSuppliers(), args)
		require.NoError(t, err)
		var resp supplier.QueryAllSuppliersResponse
		require.NoError(t, net.Config.Codec.UnmarshalJSON(out.Bytes(), &resp))
		require.NoError(t, err)
		require.Equal(t, len(suppliers), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(suppliers),
			nullify.Fill(resp.Supplier),
		)
	})
}

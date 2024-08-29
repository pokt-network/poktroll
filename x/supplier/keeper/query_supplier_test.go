package keeper_test

import (
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestSupplierQuerySingle(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	suppliers := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 2)
	tests := []struct {
		desc        string
		request     *types.QueryGetSupplierRequest
		response    *types.QueryGetSupplierResponse
		expectedErr error
	}{
		{
			desc: "First",
			request: &types.QueryGetSupplierRequest{
				OperatorAddress: suppliers[0].OperatorAddress,
			},
			response: &types.QueryGetSupplierResponse{Supplier: suppliers[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetSupplierRequest{
				OperatorAddress: suppliers[1].OperatorAddress,
			},
			response: &types.QueryGetSupplierResponse{Supplier: suppliers[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetSupplierRequest{
				OperatorAddress: strconv.Itoa(100000),
			},
			expectedErr: status.Error(codes.NotFound, "supplier with address \"100000\""),
		},
		{
			desc:        "InvalidRequest",
			expectedErr: status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			response, err := supplierModuleKeepers.Supplier(ctx, test.request)
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(test.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestSupplierQueryPaginated(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	msgs := createNSuppliers(*supplierModuleKeepers.Keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllSuppliersRequest {
		return &types.QueryAllSuppliersRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := supplierModuleKeepers.AllSuppliers(ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Supplier), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Supplier),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := supplierModuleKeepers.AllSuppliers(ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Supplier), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Supplier),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := supplierModuleKeepers.AllSuppliers(ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Supplier),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := supplierModuleKeepers.AllSuppliers(ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

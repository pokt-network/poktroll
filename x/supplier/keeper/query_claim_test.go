package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func TestClaim_QuerySingle(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	wctx := sdk.WrapSDKContext(ctx)
	claims := createNClaims(keeper, ctx, 2)
	tests := []struct {
		desc string

		request *types.QueryGetClaimRequest

		response *types.QueryGetClaimResponse
		err      error
	}{
		{
			desc: "First Claim",

			request: &types.QueryGetClaimRequest{
				SessionId:       claims[0].SessionId,
				SupplierAddress: claims[0].SupplierAddress,
			},

			response: &types.QueryGetClaimResponse{Claim: claims[0]},
			err:      nil,
		},
		{
			desc: "Second Claim",

			request: &types.QueryGetClaimRequest{
				SessionId:       claims[1].SessionId,
				SupplierAddress: claims[1].SupplierAddress,
			},

			response: &types.QueryGetClaimResponse{Claim: claims[1]},
			err:      nil,
		},
		{
			desc: "Claim Not Found - Random SessionId",

			request: &types.QueryGetClaimRequest{
				SessionId:       "not a real session id",
				SupplierAddress: claims[0].SupplierAddress,
			},

			err: status.Error(codes.NotFound, "claim not found"),
		},
		{
			desc: "Claim Not Found - Random Supplier Address",

			request: &types.QueryGetClaimRequest{
				SessionId:       claims[0].SessionId,
				SupplierAddress: sample.AccAddress(),
			},

			err: status.Error(codes.NotFound, "claim not found"),
		},
		{
			desc: "InvalidRequest - Missing SessionId",
			request: &types.QueryGetClaimRequest{
				// SessionId:       Intentionally Omitted
				SupplierAddress: claims[0].SupplierAddress,
			},

			err: types.ErrSupplierInvalidSessionId,
		},
		{
			desc: "InvalidRequest - Missing SupplierAddress",
			request: &types.QueryGetClaimRequest{
				SessionId: claims[0].SessionId,
				// SupplierAddress: Intentionally Omitted,
			},

			err: types.ErrSupplierInvalidAddress,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Claim(wctx, tc.request)
			if tc.err != nil {
				require.ErrorContains(t, err, tc.err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(tc.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestClaim_QueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	wctx := sdk.WrapSDKContext(ctx)
	claims := createNClaims(keeper, ctx, 10)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllClaimsRequest {
		return &types.QueryAllClaimsRequest{
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
		for i := 0; i < len(claims); i += step {
			resp, err := keeper.AllClaims(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
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
			resp, err := keeper.AllClaims(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Claim), step)
			require.Subset(t,
				nullify.Fill(claims),
				nullify.Fill(resp.Claim),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.AllClaims(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(claims), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(claims),
			nullify.Fill(resp.Claim),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.AllClaims(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})

	t.Run("BySupplierAddress", func(t *testing.T) {
		req := request(nil, 0, 0, true)
		req.Filter = &types.QueryAllClaimsRequest_SupplierAddress{
			SupplierAddress: claims[0].SupplierAddress,
		}
		resp, err := keeper.AllClaims(wctx, req)
		require.NoError(t, err)
		require.Equal(t, 1, int(resp.Pagination.Total))
	})

	t.Run("BySessionId", func(t *testing.T) {
		req := request(nil, 0, 0, true)
		req.Filter = &types.QueryAllClaimsRequest_SessionId{
			SessionId: claims[0].SessionId,
		}
		resp, err := keeper.AllClaims(wctx, req)
		require.NoError(t, err)
		require.Equal(t, 1, int(resp.Pagination.Total))
	})

	t.Run("BySessionEndHeight", func(t *testing.T) {
		req := request(nil, 0, 0, true)
		req.Filter = &types.QueryAllClaimsRequest_SessionEndHeight{
			SessionEndHeight: claims[0].SessionEndBlockHeight,
		}
		resp, err := keeper.AllClaims(wctx, req)
		require.NoError(t, err)
		require.Equal(t, 1, int(resp.Pagination.Total))
	})
}

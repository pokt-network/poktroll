package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestProofQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNProof(keeper, ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetProofRequest
		response *types.QueryGetProofResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetProofRequest{
				Index: msgs[0].Index,
			},
			response: &types.QueryGetProofResponse{Proof: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetProofRequest{
				Index: msgs[1].Index,
			},
			response: &types.QueryGetProofResponse{Proof: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetProofRequest{
				Index: strconv.Itoa(100000),
			},
			err: status.Error(codes.NotFound, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Proof(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
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

func TestProofQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNProof(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllProofRequest {
		return &types.QueryAllProofRequest{
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
			resp, err := keeper.ProofAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Proof), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Proof),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.ProofAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Proof), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Proof),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.ProofAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Proof),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.ProofAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

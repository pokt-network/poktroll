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
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/proof/types"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestClaimQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.ProofKeeper(t)
	claims := createNClaims(keeper, ctx, 2)

	var wrongSupplierAddr = sample.AccAddress()
	tests := []struct {
		desc string

		request *types.QueryGetClaimRequest

		response    *types.QueryGetClaimResponse
		expectedErr error
	}{
		{
			desc: "First Claim",

			request: &types.QueryGetClaimRequest{
				SessionId:       claims[0].GetSessionHeader().GetSessionId(),
				SupplierAddress: claims[0].SupplierAddress,
			},

			response:    &types.QueryGetClaimResponse{Claim: claims[0]},
			expectedErr: nil,
		},
		{
			desc: "Second Claim",

			request: &types.QueryGetClaimRequest{
				SessionId:       claims[1].GetSessionHeader().GetSessionId(),
				SupplierAddress: claims[1].SupplierAddress,
			},

			response:    &types.QueryGetClaimResponse{Claim: claims[1]},
			expectedErr: nil,
		},
		{
			desc: "Claim Not Found - Random SessionId",

			request: &types.QueryGetClaimRequest{
				SessionId:       "not a real session id",
				SupplierAddress: claims[0].GetSupplierAddress(),
			},

			expectedErr: status.Error(
				codes.NotFound,
				types.ErrProofClaimNotFound.Wrapf(
					// TODO_CONSIDERATION: factor out error message format strings to constants.
					"session ID %q and supplier %q",
					"not a real session id",
					claims[0].GetSupplierAddress(),
				).Error(),
			),
		},
		{
			desc: "Claim Not Found - Wrong Supplier Address",

			request: &types.QueryGetClaimRequest{
				SessionId:       claims[0].GetSessionHeader().GetSessionId(),
				SupplierAddress: wrongSupplierAddr,
			},

			expectedErr: status.Error(
				codes.NotFound,
				types.ErrProofClaimNotFound.Wrapf(
					"session ID %q and supplier %q",
					claims[0].GetSessionHeader().GetSessionId(),
					wrongSupplierAddr,
				).Error(),
			),
		},
		{
			desc: "InvalidRequest - Missing SessionId",
			request: &types.QueryGetClaimRequest{
				// SessionId explicitly omitted
				SupplierAddress: claims[0].GetSupplierAddress(),
			},

			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrProofInvalidSessionId.Wrap(
					"invalid empty session ID for claim being retrieved",
				).Error(),
			),
		},
		{
			desc: "InvalidRequest - Missing SupplierAddress",
			request: &types.QueryGetClaimRequest{
				SessionId: claims[0].GetSessionHeader().GetSessionId(),
				// SupplierAddress explicitly omitted
			},

			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrProofInvalidAddress.Wrap(
					"invalid supplier address for claim being retrieved ; (empty address string is not allowed)",
				).Error(),
			),
		},
		{
			desc:    "InvalidRequest - nil QueryGetClaimRequest",
			request: nil,

			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrProofInvalidQueryRequest.Wrap(
					"request cannot be nil",
				).Error(),
			),
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			response, err := keeper.Claim(ctx, test.request)
			if test.expectedErr != nil {
				actualStatus, ok := status.FromError(err)
				require.True(t, ok)

				require.ErrorIs(t, actualStatus.Err(), test.expectedErr)
				require.ErrorContains(t, err, test.expectedErr.Error())
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

func TestClaimQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.ProofKeeper(t)
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
			resp, err := keeper.AllClaims(ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Claims), step)
			require.Subset(t,
				nullify.Fill(claims),
				nullify.Fill(resp.Claims),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(claims); i += step {
			resp, err := keeper.AllClaims(ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Claims), step)
			require.Subset(t,
				nullify.Fill(claims),
				nullify.Fill(resp.Claims),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.AllClaims(ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(claims), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(claims),
			nullify.Fill(resp.Claims),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.AllClaims(ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})

	t.Run("BySupplierAddress", func(t *testing.T) {
		req := request(nil, 0, 0, true)
		req.Filter = &types.QueryAllClaimsRequest_SupplierAddress{
			SupplierAddress: claims[0].SupplierAddress,
		}
		resp, err := keeper.AllClaims(ctx, req)
		require.NoError(t, err)
		require.Equal(t, 1, int(resp.Pagination.Total))
	})

	t.Run("BySessionId", func(t *testing.T) {
		req := request(nil, 0, 0, true)
		req.Filter = &types.QueryAllClaimsRequest_SessionId{
			SessionId: claims[0].GetSessionHeader().GetSessionId(),
		}
		resp, err := keeper.AllClaims(ctx, req)
		require.NoError(t, err)
		require.Equal(t, 1, int(resp.Pagination.Total))
	})

	t.Run("BySessionEndHeight", func(t *testing.T) {
		req := request(nil, 0, 0, true)
		req.Filter = &types.QueryAllClaimsRequest_SessionEndHeight{
			SessionEndHeight: uint64(claims[0].GetSessionHeader().GetSessionEndBlockHeight()),
		}
		resp, err := keeper.AllClaims(ctx, req)
		require.NoError(t, err)
		require.Equal(t, 1, int(resp.Pagination.Total))
	})
}

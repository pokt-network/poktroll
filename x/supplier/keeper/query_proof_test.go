package keeper_test

import (
	"strconv"
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

// Prevent strconv unused error
var _ = strconv.IntSize

func TestProofQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	wctx := sdk.WrapSDKContext(ctx)
	proofs := createNProofs(keeper, ctx, 2)

	var randSupplierAddr = sample.AccAddress()
	tests := []struct {
		desc        string
		request     *types.QueryGetProofRequest
		response    *types.QueryGetProofResponse
		expectedErr error
	}{
		{
			desc: "First",
			request: &types.QueryGetProofRequest{
				SessionId:       proofs[0].GetSessionHeader().GetSessionId(),
				SupplierAddress: proofs[0].SupplierAddress,
			},
			response: &types.QueryGetProofResponse{Proof: proofs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetProofRequest{
				SessionId:       proofs[1].GetSessionHeader().GetSessionId(),
				SupplierAddress: proofs[1].SupplierAddress,
			},
			response: &types.QueryGetProofResponse{Proof: proofs[1]},
		},
		{
			desc: "Proof Not Found - Random SessionId",
			request: &types.QueryGetProofRequest{
				SessionId:       "not a real session id",
				SupplierAddress: proofs[0].GetSupplierAddress(),
			},
			expectedErr: status.Error(
				codes.NotFound,
				types.ErrSupplierProofNotFound.Wrapf(
					"session ID %q and supplier %q",
					"not a real session id",
					proofs[0].GetSupplierAddress(),
				).Error(),
			),
		},
		{
			desc: "Proof Not Found - Random Supplier Address",
			request: &types.QueryGetProofRequest{
				SessionId:       proofs[0].GetSessionHeader().GetSessionId(),
				SupplierAddress: randSupplierAddr,
			},
			expectedErr: status.Error(
				codes.NotFound,
				types.ErrSupplierProofNotFound.Wrapf(
					"session ID %q and supplier %q",
					proofs[0].GetSessionHeader().GetSessionId(),
					randSupplierAddr,
				).Error(),
			),
		},
		{
			desc: "InvalidRequest - Missing SessionId",
			request: &types.QueryGetProofRequest{
				// SessionId:       Intentionally Omitted
				SupplierAddress: proofs[0].GetSupplierAddress(),
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrSupplierInvalidSessionId.Wrapf(
					"invalid session ID for proof being retrieved %s",
					"",
				).Error(),
			),
		},
		{
			desc: "InvalidRequest - Missing SupplierAddress",
			request: &types.QueryGetProofRequest{
				SessionId: proofs[0].GetSessionHeader().GetSessionId(),
				// SupplierAddress: Intentionally Omitted,
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrSupplierInvalidAddress.Wrap(
					"invalid supplier address for proof being retrieved ; (empty address string is not allowed)",
				).Error(),
			),
		},
		{
			desc:    "InvalidRequest - nil QueryGetProofRequest",
			request: nil,
			expectedErr: status.Error(
				codes.InvalidArgument,
				types.ErrSupplierInvalidQueryRequest.Wrap(
					"request cannot be nil",
				).Error(),
			),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Proof(wctx, tc.request)
			if tc.expectedErr != nil {
				actualStatus, ok := status.FromError(err)
				require.True(t, ok)

				require.ErrorIs(t, actualStatus.Err(), tc.expectedErr)
				require.ErrorContains(t, err, tc.expectedErr.Error())
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
	keeper, ctx := keepertest.SupplierKeeper(t, nil)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNProofs(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllProofsRequest {
		return &types.QueryAllProofsRequest{
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
			resp, err := keeper.AllProofs(wctx, request(nil, uint64(i), uint64(step), false))
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
			resp, err := keeper.AllProofs(wctx, request(next, 0, uint64(step), false))
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
		resp, err := keeper.AllProofs(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Proof),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.AllProofs(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

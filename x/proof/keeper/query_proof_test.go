package keeper_test

import (
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/proto/types/proof"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
	_ "github.com/pokt-network/poktroll/testutil/testpolylog"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestProofQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.ProofKeeper(t)
	proofs := createNProofs(keeper, ctx, 2)

	var randSupplierAddr = sample.AccAddress()
	tests := []struct {
		desc        string
		request     *proof.QueryGetProofRequest
		response    *proof.QueryGetProofResponse
		expectedErr error
	}{
		{
			desc: "First",
			request: &proof.QueryGetProofRequest{
				SessionId:       proofs[0].GetSessionHeader().GetSessionId(),
				SupplierAddress: proofs[0].SupplierAddress,
			},
			response: &proof.QueryGetProofResponse{Proof: proofs[0]},
		},
		{
			desc: "Second",
			request: &proof.QueryGetProofRequest{
				SessionId:       proofs[1].GetSessionHeader().GetSessionId(),
				SupplierAddress: proofs[1].SupplierAddress,
			},
			response: &proof.QueryGetProofResponse{Proof: proofs[1]},
		},
		{
			desc: "Proof Not Found - Random SessionId",
			request: &proof.QueryGetProofRequest{
				SessionId:       "not a real session id",
				SupplierAddress: proofs[0].GetSupplierAddress(),
			},
			expectedErr: status.Error(
				codes.NotFound,
				proof.ErrProofProofNotFound.Wrapf(
					"session ID %q and supplier %q",
					"not a real session id",
					proofs[0].GetSupplierAddress(),
				).Error(),
			),
		},
		{
			desc: "Proof Not Found - Random Supplier Address",
			request: &proof.QueryGetProofRequest{
				SessionId:       proofs[0].GetSessionHeader().GetSessionId(),
				SupplierAddress: randSupplierAddr,
			},
			expectedErr: status.Error(
				codes.NotFound,
				proof.ErrProofProofNotFound.Wrapf(
					"session ID %q and supplier %q",
					proofs[0].GetSessionHeader().GetSessionId(),
					randSupplierAddr,
				).Error(),
			),
		},
		{
			desc: "InvalidRequest - Missing SessionId",
			request: &proof.QueryGetProofRequest{
				// SessionId explicitly omitted
				SupplierAddress: proofs[0].GetSupplierAddress(),
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				proof.ErrProofInvalidSessionId.Wrap(
					"invalid empty session ID for proof being retrieved",
				).Error(),
			),
		},
		{
			desc: "InvalidRequest - Missing SupplierAddress",
			request: &proof.QueryGetProofRequest{
				SessionId: proofs[0].GetSessionHeader().GetSessionId(),
				// SupplierAddress explicitly omitted
			},
			expectedErr: status.Error(
				codes.InvalidArgument,
				proof.ErrProofInvalidAddress.Wrap(
					"invalid supplier address for proof being retrieved ; (empty address string is not allowed)",
				).Error(),
			),
		},
		{
			desc:    "InvalidRequest - nil QueryGetProofRequest",
			request: nil,
			expectedErr: status.Error(
				codes.InvalidArgument,
				proof.ErrProofInvalidQueryRequest.Wrap("request cannot be nil").Error(),
			),
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			response, err := keeper.Proof(ctx, test.request)
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

func TestProofQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.ProofKeeper(t)
	proofs := createNProofs(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *proof.QueryAllProofsRequest {
		return &proof.QueryAllProofsRequest{
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
		for i := 0; i < len(proofs); i += step {
			resp, err := keeper.AllProofs(ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Proofs), step)
			require.Subset(t,
				nullify.Fill(proofs),
				nullify.Fill(resp.Proofs),
			)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(proofs); i += step {
			resp, err := keeper.AllProofs(ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Proofs), step)
			require.Subset(t,
				nullify.Fill(proofs),
				nullify.Fill(resp.Proofs),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.AllProofs(ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(proofs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(proofs),
			nullify.Fill(resp.Proofs),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.AllProofs(ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

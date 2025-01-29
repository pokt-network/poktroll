package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/x/migration/types"
)

func TestMorseAccountStateQuery(t *testing.T) {
	keeper, ctx := keepertest.MigrationKeeper(t)
	item := createTestMorseAccountState(keeper, ctx)
	tests := []struct {
		desc     string
		request  *types.QueryGetMorseAccountStateRequest
		response *types.QueryGetMorseAccountStateResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetMorseAccountStateRequest{},
			response: &types.QueryGetMorseAccountStateResponse{MorseAccountState: item},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.MorseAccountState(ctx, tc.request)
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

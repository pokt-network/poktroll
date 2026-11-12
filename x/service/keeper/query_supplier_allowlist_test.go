package keeper_test

import (
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestQuerySupplierAllowlist(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	k.SetService(ctx, sharedtypes.Service{Id: "svc1", Name: "svc1"})
	k.SetService(ctx, sharedtypes.Service{Id: "svc10", Name: "svc10"})
	k.SetService(ctx, sharedtypes.Service{Id: "empty", Name: "empty"})

	addrs := []string{sample.AccAddressBech32(), sample.AccAddressBech32(), sample.AccAddressBech32()}
	for _, addr := range addrs {
		k.SetSupplierAllowlistEntry(ctx, "svc1", addr)
	}
	// An entry of another service whose ID extends "svc1" must not leak in.
	otherAddr := sample.AccAddressBech32()
	k.SetSupplierAllowlistEntry(ctx, "svc10", otherAddr)

	t.Run("paginates", func(t *testing.T) {
		page1, err := k.SupplierAllowlist(ctx, &types.QuerySupplierAllowlistRequest{
			ServiceId:  "svc1",
			Pagination: &query.PageRequest{Limit: 2},
		})
		require.NoError(t, err)
		require.True(t, page1.Permissioned)
		require.Len(t, page1.Addresses, 2)
		require.NotEmpty(t, page1.Pagination.NextKey)

		page2, err := k.SupplierAllowlist(ctx, &types.QuerySupplierAllowlistRequest{
			ServiceId:  "svc1",
			Pagination: &query.PageRequest{Key: page1.Pagination.NextKey, Limit: 2},
		})
		require.NoError(t, err)
		require.True(t, page2.Permissioned)
		require.Len(t, page2.Addresses, 1)
		require.Empty(t, page2.Pagination.NextKey)

		require.ElementsMatch(t, addrs, append(page1.Addresses, page2.Addresses...))
	})

	t.Run("lists exactly the service's addresses", func(t *testing.T) {
		res, err := k.SupplierAllowlist(ctx, &types.QuerySupplierAllowlistRequest{ServiceId: "svc10"})
		require.NoError(t, err)
		require.Equal(t, []string{otherAddr}, res.Addresses)
	})

	t.Run("empty list is not permissioned", func(t *testing.T) {
		res, err := k.SupplierAllowlist(ctx, &types.QuerySupplierAllowlistRequest{ServiceId: "empty"})
		require.NoError(t, err)
		require.False(t, res.Permissioned)
		require.Empty(t, res.Addresses)
	})

	t.Run("unknown service", func(t *testing.T) {
		_, err := k.SupplierAllowlist(ctx, &types.QuerySupplierAllowlistRequest{ServiceId: "unknown"})
		require.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("nil request", func(t *testing.T) {
		_, err := k.SupplierAllowlist(ctx, nil)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

func TestQuerySupplierAllowlistEntry(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	k.SetService(ctx, sharedtypes.Service{Id: "svc1", Name: "svc1"})
	k.SetService(ctx, sharedtypes.Service{Id: "empty", Name: "empty"})

	listed := sample.AccAddressBech32()
	absent := sample.AccAddressBech32()
	k.SetSupplierAllowlistEntry(ctx, "svc1", listed)

	tests := []struct {
		desc     string
		request  *types.QuerySupplierAllowlistEntryRequest
		expected *types.QuerySupplierAllowlistEntryResponse
		code     codes.Code
	}{
		{
			desc:     "listed",
			request:  &types.QuerySupplierAllowlistEntryRequest{ServiceId: "svc1", Address: listed},
			expected: &types.QuerySupplierAllowlistEntryResponse{Permissioned: true, Listed: true},
		},
		{
			desc:     "absent",
			request:  &types.QuerySupplierAllowlistEntryRequest{ServiceId: "svc1", Address: absent},
			expected: &types.QuerySupplierAllowlistEntryResponse{Permissioned: true, Listed: false},
		},
		{
			desc:     "service not permissioned",
			request:  &types.QuerySupplierAllowlistEntryRequest{ServiceId: "empty", Address: absent},
			expected: &types.QuerySupplierAllowlistEntryResponse{Permissioned: false, Listed: false},
		},
		{
			desc:    "non-canonical address",
			request: &types.QuerySupplierAllowlistEntryRequest{ServiceId: "svc1", Address: strings.ToUpper(listed)},
			code:    codes.InvalidArgument,
		},
		{
			desc:    "unknown service",
			request: &types.QuerySupplierAllowlistEntryRequest{ServiceId: "unknown", Address: listed},
			code:    codes.NotFound,
		},
		{
			desc: "nil request",
			code: codes.InvalidArgument,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			res, err := k.SupplierAllowlistEntry(ctx, test.request)
			if test.expected == nil {
				require.Equal(t, test.code, status.Code(err))
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expected, res)
		})
	}
}

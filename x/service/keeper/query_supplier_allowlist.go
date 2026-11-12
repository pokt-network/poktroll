package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SupplierAllowlist returns a page of the addresses on a service's supplier allowlist.
// The service is permissioned iff the allowlist is non-empty, independently of the page.
func (k Keeper) SupplierAllowlist(
	ctx context.Context,
	req *types.QuerySupplierAllowlistRequest,
) (*types.QuerySupplierAllowlistResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if _, found := k.GetService(ctx, req.ServiceId); !found {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("service ID not found: %q", req.ServiceId))
	}

	// The prefix store strips the service prefix, so the key is exactly the address.
	serviceStore := prefix.NewStore(
		k.getSupplierAllowlistStore(ctx),
		types.SupplierAllowlistServicePrefix(req.ServiceId),
	)

	var addresses []string
	pageRes, err := query.Paginate(serviceStore, req.Pagination, func(key, _ []byte) error {
		addresses = append(addresses, string(key))
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QuerySupplierAllowlistResponse{
		Permissioned: k.IsServicePermissioned(ctx, req.ServiceId),
		Addresses:    addresses,
		Pagination:   pageRes,
	}, nil
}

// SupplierAllowlistEntry reports whether an address is on a service's supplier
// allowlist, and whether the service is permissioned.
func (k Keeper) SupplierAllowlistEntry(
	ctx context.Context,
	req *types.QuerySupplierAllowlistEntryRequest,
) (*types.QuerySupplierAllowlistEntryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// The allowlist is string-keyed: a non-canonical encoding can never match an
	// entry, so reject it rather than report a misleading listed=false.
	if err := sharedtypes.ValidateCanonicalAddress(req.Address); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if _, found := k.GetService(ctx, req.ServiceId); !found {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("service ID not found: %q", req.ServiceId))
	}

	return &types.QuerySupplierAllowlistEntryResponse{
		Permissioned: k.IsServicePermissioned(ctx, req.ServiceId),
		Listed:       k.HasSupplierAllowlistEntry(ctx, req.ServiceId, req.Address),
	}, nil
}

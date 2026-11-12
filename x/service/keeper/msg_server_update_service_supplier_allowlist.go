package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/service/types"
)

// UpdateServiceSupplierAllowlist adds and removes the suppliers a service's owner
// allows to serve it. A service with a non-empty allowlist is permissioned.
//
// Allowlist changes never touch staked suppliers directly. When a change can
// make a staked supplier ineligible, a supplier sweep is scheduled for the
// service, and the supplier module deactivates the ineligible suppliers from the
// next session start:
//   - the service just became permissioned: an initial sweep, deferred so the
//     owner can finish building the list first;
//   - an entry was really removed from a list that stays non-empty: a sweep
//     request, which never restarts a running sweep (it only asks for one more
//     pass once the running one ends).
//
// Adding entries alone never makes a supplier ineligible, so it neither
// schedules nor touches a sweep.
func (k msgServer) UpdateServiceSupplierAllowlist(
	goCtx context.Context,
	msg *types.MsgUpdateServiceSupplierAllowlist,
) (*types.MsgUpdateServiceSupplierAllowlistResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger().With("method", "UpdateServiceSupplierAllowlist")
	logger.Info(fmt.Sprintf(
		"About to update the supplier allowlist of service %q (add: %d, remove: %d)",
		msg.ServiceId, len(msg.Add), len(msg.Remove),
	))

	// ValidateBasic is what guarantees every address is canonical: the allowlist
	// is string-keyed, and the keeper methods below do not validate addresses.
	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	foundService, found := k.GetService(ctx, msg.ServiceId)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			types.ErrServiceNotFound.Wrapf("service %q not found", msg.ServiceId).Error(),
		)
	}

	if foundService.OwnerAddress != msg.OwnerAddress {
		return nil, status.Error(
			codes.PermissionDenied,
			types.ErrServiceUnauthorized.Wrapf(
				"signer %q is not the owner of service %q (owner: %q)",
				msg.OwnerAddress, msg.ServiceId, foundService.OwnerAddress,
			).Error(),
		)
	}

	wasPermissioned := k.IsServicePermissioned(ctx, msg.ServiceId)

	// Removes apply before adds; only the entries that really changed are reported.
	added, removed := k.UpdateSupplierAllowlistEntries(ctx, msg.ServiceId, msg.Add, msg.Remove)

	isPermissioned := k.IsServicePermissioned(ctx, msg.ServiceId)

	switch {
	case !wasPermissioned && isPermissioned:
		k.ScheduleInitialAllowlistSweep(ctx, msg.ServiceId)
	case wasPermissioned && isPermissioned && len(removed) > 0:
		k.RequestAllowlistSweep(ctx, msg.ServiceId)
	case wasPermissioned && !isPermissioned:
		// The restriction is gone: no supplier needs to be swept.
		k.ClearAllowlistSweep(ctx, msg.ServiceId)
	}

	event := &types.EventServiceSupplierAllowlistUpdated{
		ServiceId:    msg.ServiceId,
		OwnerAddress: msg.OwnerAddress,
		Permissioned: isPermissioned,
		Added:        added,
		Removed:      removed,
	}
	if err := ctx.EventManager().EmitTypedEvent(event); err != nil {
		err = types.ErrServiceEmitEvent.Wrapf("(%+v): %s", event, err)
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	logger.Info(fmt.Sprintf(
		"Updated the supplier allowlist of service %q (added: %d, removed: %d, permissioned: %t)",
		msg.ServiceId, len(added), len(removed), isPermissioned,
	))

	return &types.MsgUpdateServiceSupplierAllowlistResponse{}, nil
}

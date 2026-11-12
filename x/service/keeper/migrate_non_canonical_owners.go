package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// CanonicalizeServiceOwnerAddresses rewrites every service owner address stored
// in a non-canonical bech32 encoding (accepted before MsgAddService required the
// canonical form) to its canonical form, so the owner can still update and
// transfer the service. Services are keyed by ID, so no key changes.
func (k Keeper) CanonicalizeServiceOwnerAddresses(ctx context.Context) (numCanonicalized int) {
	logger := k.Logger().With("method", "CanonicalizeServiceOwnerAddresses")

	for _, service := range k.GetAllServices(ctx) {
		ownerAddress, err := cosmostypes.AccAddressFromBech32(service.OwnerAddress)
		if err != nil || ownerAddress.String() == service.OwnerAddress {
			continue
		}
		logger.Info(fmt.Sprintf("canonicalized owner address %s of service %s", service.OwnerAddress, service.Id))
		service.OwnerAddress = ownerAddress.String()
		k.SetService(ctx, service)
		numCanonicalized++
	}

	return numCanonicalized
}

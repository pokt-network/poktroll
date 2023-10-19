package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"pocket/x/session/types"
	sharedtypes "pocket/x/shared/types"
)

// TODO(@Olshansk): Need to properly scaffold this read tx
func (k Keeper) GetSession(
	ctx sdk.Context,
	appAddress string,
	serviceId *sharedtypes.ServiceId,
	blockHeight int64,
) (*types.Session, error) {
	logger := k.Logger(ctx).With("method", "GetSession")
	sessionHydrator := NewSessionHydrator(logger, appAddress, serviceId, blockHeight, k, ctx)
	return sessionHydrator.hydrateSession()
}

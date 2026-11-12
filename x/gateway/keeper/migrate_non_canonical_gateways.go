package keeper

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

// MigrateNonCanonicalGateways begins unbonding every gateway stored under a
// non-canonical bech32 address (accepted before gateway messages required the
// canonical form), as MsgUnstakeGateway would. No message can address such a
// gateway anymore, so without this its stake could never be released. The refund
// goes to the same account bytes, and applications delegated to it are
// auto-undelegated once it is inactive.
//
// Addresses that do not decode are skipped: unbonding refunds to the decoded
// address, so marking them unbonding would fail the unbonding EndBlocker.
// Every key written is the stored string, so a gateway stored under the
// canonical encoding of the same account is never touched.
func (k Keeper) MigrateNonCanonicalGateways(ctx context.Context) (numUnbondingBegun int) {
	logger := k.Logger().With("method", "MigrateNonCanonicalGateways")
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sessionEndHeight := k.sharedKeeper.GetSessionEndHeight(ctx, sdkCtx.BlockHeight())

	// GetAllGateways collects every gateway, card included, before the loop below writes.
	for _, gateway := range k.GetAllGateways(ctx) {
		accAddress, err := cosmostypes.AccAddressFromBech32(gateway.Address)
		if err != nil || accAddress.String() == gateway.Address || gateway.IsUnbonding() {
			continue
		}
		gateway.UnstakeSessionEndHeight = uint64(sessionEndHeight)
		k.SetGateway(ctx, gateway)
		numUnbondingBegun++
		logger.Info(fmt.Sprintf("began unbonding gateway with non-canonical address %s", gateway.Address))

		unbondingBeginEvent := &types.EventGatewayUnbondingBegin{
			Gateway:            gateway.DehydratedForEvent(),
			SessionEndHeight:   sessionEndHeight,
			UnbondingEndHeight: k.GetGatewayUnbondingEndHeight(ctx, &gateway),
		}
		if err := sdkCtx.EventManager().EmitTypedEvent(unbondingBeginEvent); err != nil {
			logger.Error(fmt.Sprintf("failed to emit %T for gateway %s: %s", unbondingBeginEvent, gateway.Address, err))
		}
	}

	return numUnbondingBegun
}

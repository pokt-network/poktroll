package keeper

import (
	"context"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// SettleSessionAccounting is responsible for all of the post-session accounting
// necessary to burn, mint or transfer tokens depending on the amount of work
// done. The amount of "work done" complete is dictated by `sum` of `root`.
//
// ASSUMPTION: It is assumed the caller of this function validated the claim
// against a proof BEFORE calling this function.
//
// TODO_BLOCKER(@Olshansk): Is there a way to limit who can call this function?
// TODO_UPNEXT(#323, @Olshansk): Implement this function
func (k Keeper) SettleSessionAccounting(
	goCtx context.Context,
	claim *suppliertypes.Claim,
) error {
	return nil
}

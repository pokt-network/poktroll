package keeper

import (
	"context"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// SettleSessionAccounting implements TokenomicsKeeper#SettleSessionAccounting
func (k TokenomicsKeeper) SettleSessionAccounting(
	goCtx context.Context,
	claim *suppliertypes.Claim,
) error {
	return nil
}

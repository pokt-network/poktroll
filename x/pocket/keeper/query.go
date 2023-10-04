package keeper

import (
	"pocket/x/pocket/types"
)

var _ types.QueryServer = Keeper{}

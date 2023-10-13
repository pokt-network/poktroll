package keeper

import (
	"pocket/x/gateway/types"
)

var _ types.QueryServer = Keeper{}

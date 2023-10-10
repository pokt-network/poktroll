package keeper

import (
	"pocket/x/service/types"
)

var _ types.QueryServer = Keeper{}

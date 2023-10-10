package keeper

import (
	"pocket/x/application/types"
)

var _ types.QueryServer = Keeper{}

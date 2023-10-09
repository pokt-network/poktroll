package keeper

import (
	"pocket/x/session/types"
)

var _ types.QueryServer = Keeper{}

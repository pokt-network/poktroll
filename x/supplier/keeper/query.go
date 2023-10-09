package keeper

import (
	"pocket/x/supplier/types"
)

var _ types.QueryServer = Keeper{}

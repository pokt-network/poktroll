package keeper

import (
	"github.com/pokt-network/poktroll/x/pocket/types"
)

var _ types.QueryServer = Keeper{}

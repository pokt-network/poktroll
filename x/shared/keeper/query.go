package keeper

import (
	"github.com/pokt-network/pocket/x/shared/types"
)

var _ types.QueryServer = Keeper{}

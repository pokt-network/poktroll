package keeper

import (
	"github.com/pokt-network/pocket/x/migration/types"
)

var _ types.QueryServer = Keeper{}

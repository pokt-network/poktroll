package keeper

import (
	"github.com/pokt-network/poktroll/x/migration/types"
)

var _ types.QueryServer = Keeper{}

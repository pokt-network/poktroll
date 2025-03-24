package keeper

import (
	"github.com/pokt-network/poktroll/x/shared/types"
)

var _ types.QueryServer = Keeper{}

package keeper

import (
	"github.com/pokt-network/poktroll/x/paramgov/types"
)

var _ types.QueryServer = Keeper{}

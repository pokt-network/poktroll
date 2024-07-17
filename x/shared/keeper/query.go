package keeper

import (
	"github.com/pokt-network/poktroll/proto/types/shared"
)

var _ shared.QueryServer = Keeper{}

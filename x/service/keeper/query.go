package keeper

import "github.com/pokt-network/poktroll/proto/types/service"

var _ service.QueryServer = Keeper{}

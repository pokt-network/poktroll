package keeper

import "github.com/pokt-network/poktroll/proto/types/gateway"

var _ gateway.QueryServer = Keeper{}

package keeper

import "github.com/pokt-network/poktroll/proto/types/supplier"

var _ supplier.QueryServer = Keeper{}

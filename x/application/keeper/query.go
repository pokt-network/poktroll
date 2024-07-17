package keeper

import "github.com/pokt-network/poktroll/proto/types/application"

var _ application.QueryServer = Keeper{}

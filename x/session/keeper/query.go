package keeper

import "github.com/pokt-network/poktroll/proto/types/session"

var _ session.QueryServer = Keeper{}

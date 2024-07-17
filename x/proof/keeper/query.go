package keeper

import "github.com/pokt-network/poktroll/proto/types/proof"

var _ proof.QueryServer = Keeper{}

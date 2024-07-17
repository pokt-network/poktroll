package keeper

import "github.com/pokt-network/poktroll/proto/types/tokenomics"

var _ tokenomics.QueryServer = Keeper{}

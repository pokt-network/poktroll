package keeper

import (
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

type msgServer struct {
	TokenomicsKeeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper TokenomicsKeeper) types.MsgServer {
	return &msgServer{TokenomicsKeeper: keeper}
}

var _ types.MsgServer = msgServer{}

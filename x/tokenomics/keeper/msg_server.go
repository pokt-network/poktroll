package keeper

import "github.com/pokt-network/poktroll/proto/types/tokenomics"

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) tokenomics.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ tokenomics.MsgServer = msgServer{}

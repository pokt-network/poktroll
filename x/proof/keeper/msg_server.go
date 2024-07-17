package keeper

import "github.com/pokt-network/poktroll/proto/types/proof"

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) proof.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ proof.MsgServer = msgServer{}

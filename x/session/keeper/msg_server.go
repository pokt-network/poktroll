package keeper

import "github.com/pokt-network/poktroll/proto/types/session"

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) session.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ session.MsgServer = msgServer{}

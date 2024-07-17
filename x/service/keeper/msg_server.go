package keeper

import "github.com/pokt-network/poktroll/proto/types/service"

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) service.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ service.MsgServer = msgServer{}

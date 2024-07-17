package keeper

import "github.com/pokt-network/poktroll/proto/types/application"

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) application.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ application.MsgServer = msgServer{}

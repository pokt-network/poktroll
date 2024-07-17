package keeper

import "github.com/pokt-network/poktroll/proto/types/gateway"

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) gateway.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ gateway.MsgServer = msgServer{}

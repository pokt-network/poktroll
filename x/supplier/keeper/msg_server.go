package keeper

import "github.com/pokt-network/poktroll/proto/types/supplier"

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) supplier.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ supplier.MsgServer = msgServer{}

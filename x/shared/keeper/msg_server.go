package keeper

import (
	"github.com/pokt-network/poktroll/proto/types/shared"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) shared.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ shared.MsgServer = msgServer{}

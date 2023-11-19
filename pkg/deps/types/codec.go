package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

var depCodec *codec.ProtoCodec

func init() {
	reg := codectypes.NewInterfaceRegistry()
	accounttypes.RegisterInterfaces(reg)
	depCodec = codec.NewProtoCodec(reg)
}

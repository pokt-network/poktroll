package client

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// QueryCodec is a codec used to unmarshal the account interface returned by the
// account querier into the concrete account interface implementation registered
// in the interface registry of the auth module
var QueryCodec *codec.ProtoCodec

func init() {
	reg := codectypes.NewInterfaceRegistry()
	accounttypes.RegisterInterfaces(reg)
	cryptocodec.RegisterInterfaces(reg)
	QueryCodec = codec.NewProtoCodec(reg)
}

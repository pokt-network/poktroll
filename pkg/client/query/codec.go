package query

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// queryCodec is a codec used to unmarshal the account interface returned by the
// account querier into the concrete account interface implementation registered
// in the interface registry of the auth module
var queryCodec *codec.ProtoCodec

func init() {
	reg := codectypes.NewInterfaceRegistry()
	accounttypes.RegisterInterfaces(reg)
	queryCodec = codec.NewProtoCodec(reg)
}

package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	// this line is used by starport scaffolding # 1
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgImportMorseClaimableAccounts{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgClaimMorseAccount{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgClaimMorseApplication{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgClaimMorseSupplier{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
	&MsgClaimMorseGateway{},
)
// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

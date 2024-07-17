package application

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	// this line is used by starport scaffolding # 1
)

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgStakeApplication{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUnstakeApplication{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDelegateToGateway{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUndelegateFromGateway{},
	)
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

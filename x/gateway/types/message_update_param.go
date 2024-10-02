package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ cosmostypes.Msg = (*MsgUpdateParam)(nil)

func NewMsgUpdateParam(authority string, name string, asType any) *MsgUpdateParam {
	var asTypeIface isMsgUpdateParam_AsType

	switch t := asType.(type) {
	case *cosmostypes.Coin:
		asTypeIface = &MsgUpdateParam_AsCoin{AsCoin: t}
	default:
		panic(fmt.Sprintf("unexpected param value type: %T", asType))
	}

	return &MsgUpdateParam{
		Authority: authority,
		Name:      name,
		AsType:    asTypeIface,
	}
}

func (msg *MsgUpdateParam) ValidateBasic() error {
	_, err := cosmostypes.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	// Parameter value cannot be nil.
	if msg.AsType == nil {
		return ErrGatewayParamInvalid.Wrap("missing param AsType")
	}

	// Parameter name must be supported by this module.
	switch msg.Name {
	case ParamMinStake:
		return msg.paramTypeIsCoin()
	default:
		return ErrGatewayParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}
}

func (msg *MsgUpdateParam) paramTypeIsCoin() error {
	_, ok := msg.AsType.(*MsgUpdateParam_AsCoin)
	if !ok {
		return ErrGatewayParamInvalid.Wrapf(
			"invalid type for param %q expected %T type: %T",
			msg.Name, &MsgUpdateParam_AsCoin{}, msg.AsType,
		)
	}

	return nil
}

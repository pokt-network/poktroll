package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ cosmostypes.Msg = (*MsgUpdateParam)(nil)

func NewMsgUpdateParam(authority string, name string, value any) *MsgUpdateParam {
	var valueAsType isMsgUpdateParam_AsType

	switch v := value.(type) {
	case *cosmostypes.Coin:
		valueAsType = &MsgUpdateParam_AsCoin{AsCoin: v}
	default:
		panic(fmt.Sprintf("unexpected param value type: %T", value))
	}

	return &MsgUpdateParam{
		Authority: authority,
		Name:      name,
		AsType:    valueAsType,
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

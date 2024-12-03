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
	case uint64:
		asTypeIface = &MsgUpdateParam_AsUint64{AsUint64: t}
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

	// Parameter value MUST NOT be nil.
	if msg.AsType == nil {
		return ErrAppParamInvalid.Wrapf("missing param AsType for parameter %q", msg.Name)
	}

	// Parameter name MUST be supported by this module.
	switch msg.Name {
	case ParamMaxDelegatedGateways:
		if err := genericParamTypeIs[*MsgUpdateParam_AsUint64](msg); err != nil {
			return err
		}
		return ValidateMaxDelegatedGateways(msg.GetAsUint64())
	case ParamMinStake:
		if err := genericParamTypeIs[*MsgUpdateParam_AsCoin](msg); err != nil {
			return err
		}
		return ValidateMinStake(msg.GetAsCoin())
	default:
		return ErrAppParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}
}

// genericTypeIsCoin checks if the parameter type is T, returning an error if not.
func genericParamTypeIs[T any](msg *MsgUpdateParam) error {
	if _, ok := msg.AsType.(T); !ok {
		return ErrAppParamInvalid.Wrapf(
			"invalid type for param %q; expected %T, got %T",
			msg.Name, *new(T), msg.AsType,
		)
	}

	return nil
}

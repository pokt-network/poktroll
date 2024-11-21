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

// ValidateBasic performs a basic validation of the MsgUpdateParam fields. It ensures:
// 1. The parameter name is supported.
// 2. The parameter type matches the expected type for a given parameter name.
// 3. The parameter value is valid (according to its respective validation function).
func (msg *MsgUpdateParam) ValidateBasic() error {
	_, err := cosmostypes.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	// Parameter value MUST NOT be nil.
	if msg.AsType == nil {
		return ErrSupplierParamInvalid.Wrap("missing param AsType")
	}

	// Parameter name MUST be supported by this module.
	switch msg.Name {
	case ParamMinStake:
		if err := genericParamTypeIs[*MsgUpdateParam_AsCoin](msg); err != nil {
			return err
		}
		return ValidateMinStake(msg.GetAsCoin())
	case ParamStakingFee:
		if err := genericParamTypeIs[*MsgUpdateParam_AsCoin](msg); err != nil {
			return err
		}
		return ValidateStakingFee(msg.GetAsCoin())
	default:
		return ErrSupplierParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}
}

func genericParamTypeIs[T any](msg *MsgUpdateParam) error {
	if _, ok := msg.AsType.(T); !ok {
		return ErrSupplierParamInvalid.Wrapf(
			"invalid type for param %q; expected %T, got %T",
			msg.Name, *new(T), msg.AsType,
		)
	}

	return nil
}

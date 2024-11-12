package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = (*MsgUpdateParam)(nil)

func NewMsgUpdateParam(authority string, name string, asType any) (*MsgUpdateParam, error) {
	var asTypeIface isMsgUpdateParam_AsType

	switch t := asType.(type) {
	case uint64:
		asTypeIface = &MsgUpdateParam_AsUint64{AsUint64: t}
	default:
		return nil, ErrSessionParamInvalid.Wrapf("unexpected param value type: %T", asType)
	}

	return &MsgUpdateParam{
		Authority: authority,
		Name:      name,
		AsType:    asTypeIface,
	}, nil
}

// ValidateBasic performs a basic validation of the MsgUpdateParam fields. It ensures:
// 1. The parameter name is supported.
// 2. The parameter type matches the expected type for a given parameter name.
// 3. The parameter value is valid (according to its respective validation function).
func (msg *MsgUpdateParam) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	switch msg.Name {
	case ParamNumSuppliersPerSession:
		if err := msg.paramTypeIsUint64(); err != nil {
			return err
		}
		return ValidateNumSuppliersPerSession(msg.GetAsUint64())
	default:
		return ErrSessionParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}
}

func (msg *MsgUpdateParam) paramTypeIsUint64() error {
	if _, ok := msg.AsType.(*MsgUpdateParam_AsUint64); !ok {
		return ErrSessionParamInvalid.Wrapf(
			"invalid type for param %q expected %T, got %T",
			msg.Name, &MsgUpdateParam_AsUint64{}, msg.AsType,
		)
	}
	return nil
}

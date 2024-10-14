package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = (*MsgUpdateParam)(nil)

// NewMsgUpdateParam creates a new MsgUpdateParam instance for a single
// governance parameter update.
func NewMsgUpdateParam(authority string, name string, asType any) (*MsgUpdateParam, error) {
	var asTypeIface isMsgUpdateParam_AsType

	switch t := asType.(type) {
	case *sdk.Coin:
		asTypeIface = &MsgUpdateParam_AsCoin{AsCoin: t}
	default:
		return nil, ErrServiceParamInvalid.Wrapf("unexpected param value type: %T", asType)
	}

	return &MsgUpdateParam{
		Authority: authority,
		Name:      name,
		AsType:    asTypeIface,
	}, nil
}

// ValidateBasic performs a basic validation of the MsgUpdateParam fields. It ensures
// the parameter name is supported and the parameter type matches the expected type for
// a given parameter name.
func (msg *MsgUpdateParam) ValidateBasic() error {
	// Validate the address
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return ErrServiceInvalidAddress.Wrapf("invalid authority address %s; (%v)", msg.Authority, err)
	}

	// Parameter value MUST NOT be nil.
	if msg.AsType == nil {
		return ErrServiceParamInvalid.Wrap("missing param AsType")
	}

	// Parameter name MUST be supported by this module.
	switch msg.Name {
	case ParamAddServiceFee:
		if err := msg.paramTypeIsCoin(); err != nil {
			return err
		}
		return ValidateAddServiceFee(msg.GetAsCoin())
	default:
		return ErrServiceParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}
}

// paramTypeIsCoin checks if the parameter type is *cosmostypes.Coin, returning an error if not.
func (msg *MsgUpdateParam) paramTypeIsCoin() error {
	if _, ok := msg.AsType.(*MsgUpdateParam_AsCoin); !ok {
		return ErrServiceParamInvalid.Wrapf(
			"invalid type for param %q expected %T, got %T",
			msg.Name, &MsgUpdateParam_AsCoin{},
			msg.AsType,
		)
	}
	return nil
}

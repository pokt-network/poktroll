package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = (*MsgUpdateParam)(nil)

func NewMsgUpdateParam(authority string, name string, asTypeAny any) (*MsgUpdateParam, error) {
	var asTypeIface isMsgUpdateParam_AsType

	switch asType := asTypeAny.(type) {
	case float64:
		asTypeIface = &MsgUpdateParam_AsFloat{AsFloat: asType}
	default:
		return nil, fmt.Errorf("unexpected param value type: %T", asTypeAny)
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
		return ErrTokenomicsAddressInvalid.Wrapf("invalid authority address %s; (%v)", msg.Authority, err)
	}

	// Parameter value cannot be nil.
	if msg.AsType == nil {
		return ErrTokenomicsParamsInvalid.Wrap("missing param AsType")
	}

	// Parameter name must be supported by this module.
	switch msg.Name {
	case ParamMintAllocationDao:
		if err := msg.paramTypeIsFloat(); err != nil {
			return err
		}
		return ValidateMintAllocationDao(msg.GetAsFloat())
	case ParamMintAllocationProposer:
		if err := msg.paramTypeIsFloat(); err != nil {
			return err
		}
		return ValidateMintAllocationProposer(msg.GetAsFloat())
	case ParamMintAllocationSupplier:
		if err := msg.paramTypeIsFloat(); err != nil {
			return err
		}
		return ValidateMintAllocationSupplier(msg.GetAsFloat())
	case ParamMintAllocationSourceOwner:
		if err := msg.paramTypeIsFloat(); err != nil {
			return err
		}
		return ValidateMintAllocationSourceOwner(msg.GetAsFloat())
	case ParamMintAllocationApplication:
		if err := msg.paramTypeIsFloat(); err != nil {
			return err
		}
		return ValidateMintAllocationApplication(msg.GetAsFloat())
	default:
		return ErrTokenomicsParamNameInvalid.Wrapf("unsupported param %q", msg.Name)
	}
}

func (msg *MsgUpdateParam) paramTypeIsFloat() error {
	if _, ok := msg.AsType.(*MsgUpdateParam_AsFloat); !ok {
		return ErrTokenomicsParamInvalid.Wrapf(
			"invalid type for param %q; expected %T, got %T",
			msg.Name, &MsgUpdateParam_AsFloat{}, msg.AsType,
		)
	}

	return nil
}

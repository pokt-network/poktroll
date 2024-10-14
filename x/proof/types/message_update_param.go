package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = (*MsgUpdateParam)(nil)

// NewMsgUpdateParam creates a new MsgUpdateParam instance for a single
// governance parameter update.
func NewMsgUpdateParam(authority string, name string, value any) (*MsgUpdateParam, error) {
	var valueAsType isMsgUpdateParam_AsType

	switch v := value.(type) {
	case string:
		valueAsType = &MsgUpdateParam_AsString{AsString: v}
	case int64:
		valueAsType = &MsgUpdateParam_AsInt64{AsInt64: v}
	case []byte:
		valueAsType = &MsgUpdateParam_AsBytes{AsBytes: v}
	case *sdk.Coin:
		valueAsType = &MsgUpdateParam_AsCoin{AsCoin: v}
	default:
		return nil, fmt.Errorf("unexpected param value type: %T", value)
	}

	return &MsgUpdateParam{
		Authority: authority,
		Name:      name,
		AsType:    valueAsType,
	}, nil
}

// ValidateBasic performs a basic validation of the MsgUpdateParam fields. It ensures:
// 1. The parameter name is supported.
// 2. The parameter type matches the expected type for a given parameter name.
// 3. The parameter value is valid (according to its respective validation function).
func (msg *MsgUpdateParam) ValidateBasic() error {
	// Validate the address
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return ErrProofInvalidAddress.Wrapf("invalid authority address %s; (%v)", msg.Authority, err)
	}

	// Parameter value cannot be nil.
	if msg.AsType == nil {
		return ErrProofParamInvalid.Wrap("missing param AsType")
	}

	// Parameter name must be supported by this module.
	switch msg.Name {
	case ParamProofRequestProbability:
		if err := msg.paramTypeIsFloat(); err != nil {
			return err
		}
		return ValidateProofRequestProbability(msg.GetAsFloat())
	case ParamProofRequirementThreshold:
		if err := msg.paramTypeIsCoin(); err != nil {
			return err
		}
		return ValidateProofRequirementThreshold(msg.GetAsCoin())
	case ParamProofMissingPenalty:
		if err := msg.paramTypeIsCoin(); err != nil {
			return err
		}
		return ValidateProofMissingPenalty(msg.GetAsCoin())
	case ParamProofSubmissionFee:
		if err := msg.paramTypeIsCoin(); err != nil {
			return err
		}
		return ValidateProofSubmissionFee(msg.GetAsCoin())
	default:
		return ErrProofParamNameInvalid.Wrapf("unsupported param %q", msg.Name)
	}
}

// paramTypeIsFloat checks if the parameter type is Float, returning an error if not.
func (msg *MsgUpdateParam) paramTypeIsFloat() error {
	if _, ok := msg.AsType.(*MsgUpdateParam_AsFloat); !ok {
		return ErrProofParamInvalid.Wrapf(
			"invalid type for param %q expected %T, got %T",
			msg.Name, &MsgUpdateParam_AsFloat{},
			msg.AsType,
		)
	}
	return nil
}

// paramTypeIsCoin checks if the parameter type is *cosmostypes.Coin, returning an error if not.
func (msg *MsgUpdateParam) paramTypeIsCoin() error {
	if _, ok := msg.AsType.(*MsgUpdateParam_AsCoin); !ok {
		return ErrProofParamInvalid.Wrapf(
			"invalid type for param %q expected %T, got %T",
			msg.Name, &MsgUpdateParam_AsCoin{},
			msg.AsType,
		)
	}
	return nil
}

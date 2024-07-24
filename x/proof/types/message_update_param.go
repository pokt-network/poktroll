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
	default:
		return nil, fmt.Errorf("unexpected param value type: %T", value)
	}

	return &MsgUpdateParam{
		Authority: authority,
		Name:      name,
		AsType:    valueAsType,
	}, nil
}

// ValidateBasic performs a basic validation of the MsgUpdateParam fields. It ensures
// the parameter name is supported and the parameter type matches the expected type for
// a given parameter name.
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
	case ParamRelayDifficultyTargetHash:
		return msg.paramTypeIsBytes()
	case ParamProofRequestProbability:
		return msg.paramTypeIsFloat()
	case ParamProofRequirementThreshold:
		return msg.paramTypeIsInt64()
	case ParamProofMissingPenalty:
		return msg.paramTypeIsCoin()
	default:
		return ErrProofParamNameInvalid.Wrapf("unsupported param %q", msg.Name)
	}
}

// paramTypeIsBytes checks if the parameter type is a byte slice, returning an error if not.
func (msg *MsgUpdateParam) paramTypeIsBytes() error {
	if _, ok := msg.AsType.(*MsgUpdateParam_AsBytes); !ok {
		return ErrProofParamInvalid.Wrapf(
			"invalid type for param %q expected %T, got %T",
			msg.Name, &MsgUpdateParam_AsBytes{},
			msg.AsType,
		)
	}
	return nil
}

// paramTypeIsInt64 checks if the parameter type is int64, returning an error if not.
func (msg *MsgUpdateParam) paramTypeIsInt64() error {
	if _, ok := msg.AsType.(*MsgUpdateParam_AsInt64); !ok {
		return ErrProofParamInvalid.Wrapf(
			"invalid type for param %q expected %T, got %T",
			msg.Name, &MsgUpdateParam_AsInt64{},
			msg.AsType,
		)
	}
	return nil
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

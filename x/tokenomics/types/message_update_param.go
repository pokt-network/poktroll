package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = (*MsgUpdateParam)(nil)

func NewMsgUpdateParam(authority string, name string, asTypeAny any) (*MsgUpdateParam, error) {
	var asTypeIface isMsgUpdateParam_AsType

	switch asType := asTypeAny.(type) {
	case MintAllocationPercentages:
		asTypeIface = &MsgUpdateParam_AsMintAllocationPercentages{AsMintAllocationPercentages: &asType}
	case string:
		asTypeIface = &MsgUpdateParam_AsString{AsString: asType}
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
	case ParamMintAllocationPercentages:
		if err := genericParamTypeIs[*MsgUpdateParam_AsMintAllocationPercentages](msg); err != nil {
			return err
		}
		return ValidateMintAllocationPercentages(*msg.GetAsMintAllocationPercentages())
	case ParamDaoRewardAddress:
		if err := genericParamTypeIs[*MsgUpdateParam_AsString](msg); err != nil {
			return err
		}
		return ValidateDaoRewardAddress(msg.GetAsString())
	case ParamGlobalInflationPerClaim:
		if err := genericParamTypeIs[*MsgUpdateParam_AsFloat](msg); err != nil {
			return err
		}
		return ValidateGlobalInflationPerClaim(msg.GetAsFloat())
	default:
		return ErrTokenomicsParamNameInvalid.Wrapf("unsupported param %q", msg.Name)
	}
}

// genericTypeIsCoin checks if the parameter type is T, returning an error if not.
func genericParamTypeIs[T any](msg *MsgUpdateParam) error {
	if _, ok := msg.AsType.(T); !ok {
		return ErrTokenomicsParamInvalid.Wrapf(
			"invalid type for param %q; expected %T, got %T",
			msg.Name, *new(T), msg.AsType,
		)
	}

	return nil
}

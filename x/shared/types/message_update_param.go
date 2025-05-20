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
	case uint64:
		valueAsType = &MsgUpdateParam_AsUint64{AsUint64: v}
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

// ValidateBasic performs a basic validation of the MsgUpdateParam fields. It ensures:
// 1. The parameter name is supported.
// 2. The parameter type matches the expected type for a given parameter name.
// 3. The parameter value is valid (according to its respective validation function).
func (msg *MsgUpdateParam) ValidateBasic() error {
	// Validate the address
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return ErrSharedInvalidAddress.Wrapf("invalid authority address %s; (%v)", msg.Authority, err)
	}

	// Parameter value cannot be nil.
	if msg.AsType == nil {
		return ErrSharedParamInvalid.Wrap("missing param AsType")
	}

	// Parameter name must be supported by this module.
	switch msg.Name {
	case ParamNumBlocksPerSession:
		if err := msg.paramTypeIsUint64(); err != nil {
			return err
		}
		return ValidateNumBlocksPerSession(msg.GetAsUint64())
	case ParamGracePeriodEndOffsetBlocks:
		if err := msg.paramTypeIsUint64(); err != nil {
			return err
		}
		return ValidateGracePeriodEndOffsetBlocks(msg.GetAsUint64())
	case ParamClaimWindowOpenOffsetBlocks:
		if err := msg.paramTypeIsUint64(); err != nil {
			return err
		}
		return ValidateClaimWindowOpenOffsetBlocks(msg.GetAsUint64())
	case ParamClaimWindowCloseOffsetBlocks:
		if err := msg.paramTypeIsUint64(); err != nil {
			return err
		}
		return ValidateClaimWindowCloseOffsetBlocks(msg.GetAsUint64())
	case ParamProofWindowOpenOffsetBlocks:
		if err := msg.paramTypeIsUint64(); err != nil {
			return err
		}
		return ValidateProofWindowOpenOffsetBlocks(msg.GetAsUint64())
	case ParamProofWindowCloseOffsetBlocks:
		if err := msg.paramTypeIsUint64(); err != nil {
			return err
		}
		return ValidateProofWindowCloseOffsetBlocks(msg.GetAsUint64())
	case ParamSupplierUnbondingPeriodSessions:
		if err := msg.paramTypeIsUint64(); err != nil {
			return err
		}
		return ValidateSupplierUnbondingPeriodSessions(msg.GetAsUint64())
	case ParamApplicationUnbondingPeriodSessions:
		if err := msg.paramTypeIsUint64(); err != nil {
			return err
		}
		return ValidateApplicationUnbondingPeriodSessions(msg.GetAsUint64())
	case ParamGatewayUnbondingPeriodSessions:
		if err := msg.paramTypeIsUint64(); err != nil {
			return err
		}
		return ValidateGatewayUnbondingPeriodSessions(msg.GetAsUint64())
	case ParamComputeUnitsToTokensMultiplier:
		if err := msg.paramTypeIsUint64(); err != nil {
			return err
		}
		return ValidateComputeUnitsToTokensMultiplier(msg.GetAsUint64())
	default:
		return ErrSharedParamNameInvalid.Wrapf("unsupported param %q", msg.Name)
	}
}

// paramTypeIsUint64 checks if the parameter type is int64, returning an error if not.
func (msg *MsgUpdateParam) paramTypeIsUint64() error {
	if _, ok := msg.AsType.(*MsgUpdateParam_AsUint64); !ok {
		return ErrSharedParamInvalid.Wrapf(
			"invalid type for param %q expected %T, got %T",
			msg.Name, &MsgUpdateParam_AsUint64{},
			msg.AsType,
		)
	}
	return nil
}

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
	// TODO_IMPROVE: Add a Uint64 asType instead of using int64 for uint64 params.
	case ParamNumBlocksPerSession:
		if err := msg.paramTypeIsInt64(); err != nil {
			return err
		}
		return ValidateNumBlocksPerSession(uint64(msg.GetAsInt64()))
	case ParamGracePeriodEndOffsetBlocks:
		if err := msg.paramTypeIsInt64(); err != nil {
			return err
		}
		return ValidateGracePeriodEndOffsetBlocks(uint64(msg.GetAsInt64()))
	case ParamClaimWindowOpenOffsetBlocks:
		if err := msg.paramTypeIsInt64(); err != nil {
			return err
		}
		return ValidateClaimWindowOpenOffsetBlocks(uint64(msg.GetAsInt64()))
	case ParamClaimWindowCloseOffsetBlocks:
		if err := msg.paramTypeIsInt64(); err != nil {
			return err
		}
		return ValidateClaimWindowCloseOffsetBlocks(uint64(msg.GetAsInt64()))
	case ParamProofWindowOpenOffsetBlocks:
		if err := msg.paramTypeIsInt64(); err != nil {
			return err
		}
		return ValidateProofWindowOpenOffsetBlocks(uint64(msg.GetAsInt64()))
	case ParamProofWindowCloseOffsetBlocks:
		if err := msg.paramTypeIsInt64(); err != nil {
			return err
		}
		return ValidateProofWindowCloseOffsetBlocks(uint64(msg.GetAsInt64()))
	case ParamSupplierUnbondingPeriodSessions:
		if err := msg.paramTypeIsInt64(); err != nil {
			return err
		}
		return ValidateSupplierUnbondingPeriodSessions(uint64(msg.GetAsInt64()))
	case ParamApplicationUnbondingPeriodSessions:
		if err := msg.paramTypeIsInt64(); err != nil {
			return err
		}
		return ValidateApplicationUnbondingPeriodSessions(uint64(msg.GetAsInt64()))
	case ParamComputeUnitsToTokensMultiplier:
		if err := msg.paramTypeIsInt64(); err != nil {
			return err
		}
		return ValidateComputeUnitsToTokensMultiplier(uint64(msg.GetAsInt64()))
	default:
		return ErrSharedParamNameInvalid.Wrapf("unsupported param %q", msg.Name)
	}
}

// paramTypeIsInt64 checks if the parameter type is int64, returning an error if not.
func (msg *MsgUpdateParam) paramTypeIsInt64() error {
	if _, ok := msg.AsType.(*MsgUpdateParam_AsInt64); !ok {
		return ErrSharedParamInvalid.Wrapf(
			"invalid type for param %q expected %T, got %T",
			msg.Name, &MsgUpdateParam_AsInt64{},
			msg.AsType,
		)
	}
	return nil
}

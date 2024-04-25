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

// ValidateBasic performs a basic validation of the MsgUpdateParam fields.
func (msg *MsgUpdateParam) ValidateBasic() error {
	// Validate the address
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return ErrProofInvalidAddress.Wrapf("invalid authority address %s; (%v)", msg.Authority, err)
	}

	// ValidateBasic only checks that the name of the parameter being update is valid
	// but does not check its type or value.
	switch msg.Name {
	case NameDefaultMinRelayDifficultyBits:
		return nil
	default:
		return ErrProofParamNameInvalid.Wrapf("unsupported name param %q", msg.Name)
	}
}

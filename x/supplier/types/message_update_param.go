package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ cosmostypes.Msg = &MsgUpdateParam{}

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
	// TODO_UPNEXT(@bryanchriswhite, #612): replace with min_stake param name and call validation function.
	case "":
		return nil
	default:
		return ErrSupplierParamInvalid.Wrapf("unsupported param %q", msg.Name)
	}
}

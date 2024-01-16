package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// TypeMsgSubmitProof defines the type of the message.
const TypeMsgSubmitProof = "submit_proof"

var _ sdk.Msg = (*MsgSubmitProof)(nil)

// NewMsgSubmitProof creates a new MsgSubmitProof instance form the given parameters.
func NewMsgSubmitProof(
	supplierAddress string,
	sessionHeader *sessiontypes.SessionHeader,
	proof []byte,
) *MsgSubmitProof {
	return &MsgSubmitProof{
		SupplierAddress: supplierAddress,
		SessionHeader:   sessionHeader,
		Proof:           proof,
	}
}

// Route returns the router key for this message type.
func (msg *MsgSubmitProof) Route() string {
	return RouterKey
}

// Type returns the type of the message.
func (msg *MsgSubmitProof) Type() string {
	return TypeMsgSubmitProof
}

// GetSigners retirns the signers of this message.
func (msg *MsgSubmitProof) GetSigners() []sdk.AccAddress {
	supplierAddress, err := sdk.AccAddressFromBech32(msg.SupplierAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{supplierAddress}
}

// GetSignBytes returns the signable btes of the message.
func (msg *MsgSubmitProof) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic validation on this message.
func (msg *MsgSubmitProof) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.SupplierAddress)
	if err != nil {
		return sdkerrors.Wrapf(
			ErrSupplierInvalidAddress,
			"invalid supplierAddress address (%s)", err,
		)
	}
	if err != nil {
		return sdkerrors.Wrapf(
			ErrSupplierInvalidService,
			"invalid supplierAddress address (%s)", err,
		)
	}
	return nil
}

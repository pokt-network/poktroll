package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

const TypeMsgSubmitProof = "submit_proof"

var _ sdk.Msg = (*MsgSubmitProof)(nil)

func NewMsgSubmitProof(supplierAddress string, sessionHeader *sessiontypes.SessionHeader, proof []byte) *MsgSubmitProof {
	return &MsgSubmitProof{
		SupplierAddress: supplierAddress,
		SessionHeader:   sessionHeader,
		Proof:           proof,
	}
}

func (msg *MsgSubmitProof) Route() string {
	return RouterKey
}

func (msg *MsgSubmitProof) Type() string {
	return TypeMsgSubmitProof
}

func (msg *MsgSubmitProof) GetSigners() []sdk.AccAddress {
	supplierAddress, err := sdk.AccAddressFromBech32(msg.SupplierAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{supplierAddress}
}

func (msg *MsgSubmitProof) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSubmitProof) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.SupplierAddress)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid supplierAddress address (%s)", err)
	}
	return nil
}

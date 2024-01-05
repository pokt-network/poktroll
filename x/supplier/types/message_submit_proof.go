package types

import (
	errorsmod "cosmossdk.io/errors"
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

// ValidateBasic ensures that the bech32 address strings for the supplier and
// application addresses are valid and that the proof and service ID are not empty.
//
// TODO_CONSIDERATION: additional assertions:
// * session ID is not empty
// * session end - start height == on-chain NumBlocksPerSession (for that session)
func (msg *MsgSubmitProof) ValidateBasic() error {
	var errMsg string
	_, err := sdk.AccAddressFromBech32(msg.GetSupplierAddress())
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"supplier address %q, error: %s",
			msg.GetSupplierAddress(),
			err,
		)
	}

	_, err = sdk.AccAddressFromBech32(msg.GetSessionHeader().GetApplicationAddress())
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"application address: %q, error: %s",
			msg.GetSessionHeader().GetApplicationAddress(),
			err,
		)
	}

	if msg.GetSessionHeader().GetService().GetId() == "" {
		return ErrSupplierInvalidServiceID.Wrap("proof service ID %q cannot be empty")
	}

	if len(msg.GetProof()) == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("proof cannot be empty")
	}
	if errMsg != "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, errMsg)
	}

	return nil
}

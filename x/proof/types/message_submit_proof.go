package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

var _ sdk.Msg = (*MsgSubmitProof)(nil)

func NewMsgSubmitProof(supplierAddress string, sessionHeader *sessiontypes.SessionHeader, proof []byte) *MsgSubmitProof {
	return &MsgSubmitProof{
		SupplierAddress: supplierAddress,
		SessionHeader:   sessionHeader,
		Proof:           proof,
	}
}

func (msg *MsgSubmitProof) ValidateBasic() error {
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
		return ErrProofInvalidService.Wrap("proof service ID %q cannot be empty")
	}

	if len(msg.GetProof()) == 0 {
		return ErrProofInvalidProof.Wrap("proof cannot be empty")
	}

	// TODO_BLOCKER: attempt to deserialize the proof for additional validation.

	return nil
}

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

// ValidateBasic ensures that the bech32 address strings for the supplier and
// application addresses are valid and that the proof and service ID are not empty.
//
// TODO_BETA: Call `msg.GetSessionHeader().ValidateBasic()` once its implemented
func (msg *MsgSubmitProof) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.GetSupplierAddress()); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"supplier address %q, error: %s",
			msg.GetSupplierAddress(),
			err,
		)
	}

	if _, err := sdk.AccAddressFromBech32(msg.GetSessionHeader().GetApplicationAddress()); err != nil {
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

	// TODO_MAINNET: attempt to deserialize the proof for additional validation.

	return nil
}

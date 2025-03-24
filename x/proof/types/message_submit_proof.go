package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	sessiontypes "github.com/pokt-network/pocket/x/session/types"
)

var _ sdk.Msg = (*MsgSubmitProof)(nil)

func NewMsgSubmitProof(supplierOperatorAddress string, sessionHeader *sessiontypes.SessionHeader, proof []byte) *MsgSubmitProof {
	return &MsgSubmitProof{
		SupplierOperatorAddress: supplierOperatorAddress,
		SessionHeader:           sessionHeader,
		Proof:                   proof,
	}
}

// ValidateBasic performs basic stateless validation of a MsgSubmitProof.
func (msg *MsgSubmitProof) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.GetSupplierOperatorAddress()); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"supplier operator address %q, error: %s",
			msg.GetSupplierOperatorAddress(),
			err,
		)
	}

	// Retrieve & validate the session header
	sessionHeader := msg.SessionHeader
	if sessionHeader == nil {
		return ErrProofInvalidSessionHeader.Wrapf("session header is nil")
	}

	if err := sessionHeader.ValidateBasic(); err != nil {
		return ErrProofInvalidSessionHeader.Wrapf("%s", err)
	}

	if len(msg.GetProof()) == 0 {
		return ErrProofInvalidProof.Wrap("proof cannot be empty")
	}

	// TODO_MAINNET: attempt to deserialize the proof for additional validation.

	return nil
}

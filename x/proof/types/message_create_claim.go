package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sessiontypes "github.com/pokt-network/pocket/x/session/types"
)

const TypeMsgCreateClaim = "create_claim"

var _ sdk.Msg = (*MsgCreateClaim)(nil)

func NewMsgCreateClaim(
	supplierOperatorAddr string,
	sessionHeader *sessiontypes.SessionHeader,
	rootHash []byte,
) *MsgCreateClaim {
	return &MsgCreateClaim{
		SupplierOperatorAddress: supplierOperatorAddr,
		SessionHeader:           sessionHeader,
		RootHash:                rootHash,
	}
}

// ValidateBasic performs basic stateless validation of a MsgCreateClaim.
func (msg *MsgCreateClaim) ValidateBasic() error {
	// Validate the supplier operator address
	if _, err := sdk.AccAddressFromBech32(msg.GetSupplierOperatorAddress()); err != nil {
		return ErrProofInvalidAddress.Wrapf("%s", msg.GetSupplierOperatorAddress())
	}

	// Retrieve & validate the session header
	sessionHeader := msg.SessionHeader
	if sessionHeader == nil {
		return ErrProofInvalidSessionHeader.Wrapf("session header is nil")
	}
	if err := sessionHeader.ValidateBasic(); err != nil {
		return ErrProofInvalidSessionHeader.Wrapf("invalid session header: %v", err)
	}

	// Validate the root hash
	// TODO_IMPROVE: Only checking to make sure a non-nil hash was provided for now, but we can validate the length as well.
	if len(msg.RootHash) == 0 {
		return ErrProofInvalidClaimRootHash.Wrapf("%v", msg.RootHash)
	}

	return nil
}

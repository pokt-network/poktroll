package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedhelpers "github.com/pokt-network/poktroll/x/shared/helpers"
)

const TypeMsgCreateClaim = "create_claim"

var _ sdk.Msg = (*MsgCreateClaim)(nil)

func NewMsgCreateClaim(
	supplierAddr string,
	sessionHeader *sessiontypes.SessionHeader,
	rootHash []byte,
) *MsgCreateClaim {
	return &MsgCreateClaim{
		SupplierAddress: supplierAddr,
		SessionHeader:   sessionHeader,
		RootHash:        rootHash,
	}
}

func (msg *MsgCreateClaim) ValidateBasic() error {
	// Validate the supplier address
	_, err := sdk.AccAddressFromBech32(msg.GetSupplierAddress())
	if err != nil {
		return ErrProofInvalidAddress.Wrapf("%s", msg.GetSupplierAddress())
	}

	// Retrieve & validate the session header
	sessionHeader := msg.SessionHeader
	if sessionHeader == nil {
		return ErrProofInvalidSessionStartHeight.Wrapf("%d", sessionHeader.SessionStartBlockHeight)
		// logger.Error("received a nil session header")
		// return ErrProofInvalid
	}
	if err := sessionHeader.ValidateBasic(); err != nil {
		return ErrProofInvalidSessionStartHeight.Wrapf("%d", sessionHeader.SessionStartBlockHeight)
		// logger.Error("received an invalid session header", "error", err)
		// return types.ErrTokenomicsSessionHeaderInvalid
	}

	if sessionHeader.SessionStartBlockHeight < 0 {
		return ErrProofInvalidSessionStartHeight.Wrapf("%d", sessionHeader.SessionStartBlockHeight)
	}
	if len(sessionHeader.SessionId) == 0 {
		return ErrProofInvalidSessionId.Wrapf("%s", sessionHeader.SessionId)
	}
	if !sharedhelpers.IsValidService(sessionHeader.Service) {
		return ErrProofInvalidService.Wrapf("%v", sessionHeader.Service)
	}

	// Validate the root hash
	// TODO_IMPROVE: Only checking to make sure a non-nil hash was provided for now, but we can validate the length as well.
	if len(msg.RootHash) == 0 {
		return ErrProofInvalidClaimRootHash.Wrapf("%v", msg.RootHash)
	}

	return nil
}

package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateClaim{}

func NewMsgCreateClaim(supplierAddress string, sessionHeader string, rootHash string) *MsgCreateClaim {
	return &MsgCreateClaim{
		SupplierAddress: supplierAddress,
		SessionHeader:   sessionHeader,
		RootHash:        rootHash,
	}
}

func (msg *MsgCreateClaim) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.SupplierAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid supplierAddress address (%s)", err)
	}
	return nil
}

package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgStakeSupplier{}

func NewMsgStakeSupplier(address string, stake sdk.Coin, services string) *MsgStakeSupplier {
	return &MsgStakeSupplier{
		Address:  address,
		Stake:    stake,
		Services: services,
	}
}

func (msg *MsgStakeSupplier) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address address (%s)", err)
	}
	return nil
}

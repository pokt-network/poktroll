package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = (*MsgUpdateGatewayMetadata)(nil)

func NewMsgUpdateGatewayMetadata(address string, metadata *sharedtypes.Metadata) *MsgUpdateGatewayMetadata {
	return &MsgUpdateGatewayMetadata{
		Address:  address,
		Metadata: metadata,
	}
}

// ValidateBasic performs stateless validation of the message.
//
// DEV_NOTE: Whether the gateway exists is deliberately NOT checked here. ValidateBasic
// has no keeper access; that check lives in the message server, mirroring how
// UnstakeGateway handles a not-found gateway.
func (msg *MsgUpdateGatewayMetadata) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return ErrGatewayInvalidAddress.Wrapf("invalid gateway address %s; (%v)", msg.Address, err)
	}

	// Nil metadata is valid and means "leave the stored card unchanged".
	// A non-nil card is checked for size only -- its content is never parsed onchain.
	if err := msg.Metadata.ValidateBasic(); err != nil {
		return err
	}

	return nil
}

package types

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ cosmostypes.Msg = (*MsgTransferApplication)(nil)

func NewMsgTransferApplication(srcAddr string, dstAddr string) *MsgTransferApplication {
	return &MsgTransferApplication{
		SourceAddress:      srcAddr,
		DestinationAddress: dstAddr,
	}
}

func (msg *MsgTransferApplication) ValidateBasic() error {
	if msg.GetSourceAddress() == "" {
		return ErrAppInvalidAddress.Wrap("empty source application address")
	}

	if msg.GetDestinationAddress() == "" {
		return ErrAppInvalidAddress.Wrap("empty destination application address")
	}

	srcBech32Err := sharedtypes.ValidateCanonicalAddress(msg.GetSourceAddress())
	if srcBech32Err != nil {
		return ErrAppInvalidAddress.Wrapf("invalid source application address (%s): %v", msg.GetSourceAddress(), srcBech32Err)
	}

	dstBech32Err := sharedtypes.ValidateCanonicalAddress(msg.GetDestinationAddress())
	if dstBech32Err != nil {
		return ErrAppInvalidAddress.Wrapf("invalid destination application address (%s): %v", msg.GetDestinationAddress(), dstBech32Err)
	}

	if msg.GetSourceAddress() == msg.GetDestinationAddress() {
		return ErrAppDuplicateAddress.Wrapf("source and destination application addresses are the same: %s", msg.GetSourceAddress())
	}

	return nil
}

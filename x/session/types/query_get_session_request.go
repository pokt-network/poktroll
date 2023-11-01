package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedhelpers "pocket/x/shared/helpers"
	sharedtypes "pocket/x/shared/types"
)

// NOTE: Please note that `QueryGetSessionRequest` is not a `sdk.Msg`, and is therefore not a message/request
// that will be signable or invoke a state transition. However,  Note that sdk.Msg
func NewQueryGetSessionRequest(appAddress, serviceId string, blockHeight int64) *QueryGetSessionRequest {
	return &QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		ServiceId: &sharedtypes.ServiceId{
			Id: serviceId,
		},
		BlockHeight: blockHeight,
	}
}

func (query *QueryGetSessionRequest) ValidateBasic() error {
	// Validate the application address
	if _, err := sdk.AccAddressFromBech32(query.ApplicationAddress); err != nil {
		return sdkerrors.Wrapf(ErrSessionInvalidAppAddress, "invalid app address for session being retrieved %s; (%v)", query.ApplicationAddress, err)
	}

	// Validate the Service ID
	if !sharedhelpers.IsValidService(query.ServiceId) {
		return sdkerrors.Wrapf(ErrSessionInvalidServiceId, "invalid serviceID for session being retrieved %s;", query.ServiceId)
	}

	// Validate the height for which a session is being retrieved
	if query.BlockHeight < 0 { // Note that `0` defaults to the latest height rather than genesis
		return sdkerrors.Wrapf(ErrSessionInvalidBlockHeight, "invalid block height for session being retrieved %d;", query.BlockHeight)
	}
	return nil
}

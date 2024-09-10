package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// NOTE: Please note that `QueryGetSessionRequest` is not a `sdk.Msg`, and is therefore not a message/request
// that will be signable or invoke a state transition. However, following a similar `ValidateBasic` pattern
// allows us to localize & reuse validation logic.
func NewQueryGetSessionRequest(appAddress, serviceId string, blockHeight int64) *QueryGetSessionRequest {
	return &QueryGetSessionRequest{
		ApplicationAddress: appAddress,
		Service: &sharedtypes.Service{
			Id: serviceId,
		},
		BlockHeight: blockHeight,
	}
}

func (query *QueryGetSessionRequest) ValidateBasic() error {
	// Validate the application address
	if _, err := sdk.AccAddressFromBech32(query.ApplicationAddress); err != nil {
		return ErrSessionInvalidAppAddress.Wrapf("%s", err)
	}

	// Validate the Service ID
	if err := query.Service.ValidateBasic(); err != nil {
		return ErrSessionInvalidService.Wrapf("invalid service for session being retrieved %s; %s", query.Service, err)
	}

	// Validate the height for which a session is being retrieved
	if query.BlockHeight < 0 { // Note that `0` defaults to the latest height rather than genesis
		return ErrSessionInvalidBlockHeight.Wrapf("invalid block height for session being retrieved %d;", query.BlockHeight)
	}
	return nil
}
